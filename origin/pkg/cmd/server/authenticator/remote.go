package authenticator

import (
	"crypto/x509"
	"time"

	"github.com/openshift/kubernetes/pkg/auth/authenticator"
	unversionedauthentication "github.com/openshift/kubernetes/pkg/client/clientset_generated/internalclientset/typed/authentication/internalversion"

	oauthenticator "github.com/openshift/origin/pkg/auth/authenticator"
	"github.com/openshift/origin/pkg/auth/authenticator/anonymous"
	"github.com/openshift/origin/pkg/auth/authenticator/request/bearertoken"
	"github.com/openshift/origin/pkg/auth/authenticator/request/unionrequest"
	"github.com/openshift/origin/pkg/auth/authenticator/request/x509request"
	authncache "github.com/openshift/origin/pkg/auth/authenticator/token/cache"
	authnremote "github.com/openshift/origin/pkg/auth/authenticator/token/remotetokenreview"
	"github.com/openshift/origin/pkg/auth/group"
	"github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
)

// NewRemoteAuthenticator creates an authenticator that checks the provided remote endpoint for tokens, allows any linked clientCAs to be checked, and caches
// responses as indicated.  If no authentication is possible, the user will be system:anonymous.
func NewRemoteAuthenticator(authenticationClient unversionedauthentication.TokenReviewsGetter, clientCAs *x509.CertPool, cacheTTL time.Duration, cacheSize int) (authenticator.Request, error) {
	authenticators := []oauthenticator.Request{}

	// API token auth
	var (
		tokenAuthenticator oauthenticator.Token
		err                error
	)
	// Authenticate against the remote master
	tokenAuthenticator, err = authnremote.NewAuthenticator(authenticationClient)
	if err != nil {
		return nil, err
	}
	// Cache results
	if cacheTTL > 0 && cacheSize > 0 {
		tokenAuthenticator, err = authncache.NewAuthenticator(tokenAuthenticator, cacheTTL, cacheSize)
		if err != nil {
			return nil, err
		}
	}
	authenticators = append(authenticators, bearertoken.New(tokenAuthenticator, true))

	// Client-cert auth
	if clientCAs != nil {
		opts := x509request.DefaultVerifyOptions()
		opts.Roots = clientCAs
		certauth := x509request.New(opts, x509request.SubjectToUserConversion)
		authenticators = append(authenticators, certauth)
	}

	ret := &unionrequest.Authenticator{
		// Anonymous requests will pass the token and cert checks without errors
		// Bad tokens or bad certs will produce errors, in which case we should not continue to authenticate them as "system:anonymous"
		FailOnError: true,
		Handlers: []oauthenticator.Request{
			// Add the "system:authenticated" group to users that pass token/cert authentication
			group.NewGroupAdder(unionrequest.NewUnionAuthentication(authenticators...), []string{bootstrappolicy.AuthenticatedGroup}),
			// Fall back to the "system:anonymous" user
			anonymous.NewAuthenticator(),
		},
	}

	return ret, nil
}
