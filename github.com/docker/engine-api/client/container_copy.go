package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/openshift/golang.org/x/net/context"

	"github.com/openshift/github.com/docker/engine-api/types"
)

// ContainerStatPath returns Stat information about a path inside the container filesystem.
func (cli *Client) ContainerStatPath(ctx context.Context, containerID, path string) (types.ContainerPathStat, error) {
	query := url.Values{}
	query.Set("path", filepath.ToSlash(path)) // Normalize the paths used in the API.

	urlStr := fmt.Sprintf("/containers/%s/archive", containerID)
	response, err := cli.head(ctx, urlStr, query, nil)
	if err != nil {
		return types.ContainerPathStat{}, err
	}
	defer ensureReaderClosed(response)
	return getContainerPathStatFromHeader(response.header)
}

// CopyToContainer copies content into the container filesystem.
func (cli *Client) CopyToContainer(ctx context.Context, container, path string, content io.Reader, options types.CopyToContainerOptions) error {
	query := url.Values{}
	query.Set("path", filepath.ToSlash(path)) // Normalize the paths used in the API.
	// Do not allow for an existing directory to be overwritten by a non-directory and vice versa.
	if !options.AllowOverwriteDirWithFile {
		query.Set("noOverwriteDirNonDir", "true")
	}

	apiPath := fmt.Sprintf("/containers/%s/archive", container)

	response, err := cli.putRaw(ctx, apiPath, query, content, nil)
	if err != nil {
		return err
	}
	defer ensureReaderClosed(response)

	if response.statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code from daemon: %d", response.statusCode)
	}

	return nil
}

// CopyFromContainer gets the content from the container and returns it as a Reader
// to manipulate it in the host. It's up to the caller to close the reader.
func (cli *Client) CopyFromContainer(ctx context.Context, container, srcPath string) (io.ReadCloser, types.ContainerPathStat, error) {
	query := make(url.Values, 1)
	query.Set("path", filepath.ToSlash(srcPath)) // Normalize the paths used in the API.

	apiPath := fmt.Sprintf("/containers/%s/archive", container)
	response, err := cli.get(ctx, apiPath, query, nil)
	if err != nil {
		return nil, types.ContainerPathStat{}, err
	}

	if response.statusCode != http.StatusOK {
		return nil, types.ContainerPathStat{}, fmt.Errorf("unexpected status code from daemon: %d", response.statusCode)
	}

	// In order to get the copy behavior right, we need to know information
	// about both the source and the destination. The response headers include
	// stat info about the source that we can use in deciding exactly how to
	// copy it locally. Along with the stat info about the local destination,
	// we have everything we need to handle the multiple possibilities there
	// can be when copying a file/dir from one location to another file/dir.
	stat, err := getContainerPathStatFromHeader(response.header)
	if err != nil {
		return nil, stat, fmt.Errorf("unable to get resource stat from response: %s", err)
	}
	return response.body, stat, err
}

func getContainerPathStatFromHeader(header http.Header) (types.ContainerPathStat, error) {
	var stat types.ContainerPathStat

	encodedStat := header.Get("X-Docker-Container-Path-Stat")
	statDecoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(encodedStat))

	err := json.NewDecoder(statDecoder).Decode(&stat)
	if err != nil {
		err = fmt.Errorf("unable to decode container path stat header: %s", err)
	}

	return stat, err
}
