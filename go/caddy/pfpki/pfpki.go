package pfpki

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/inverse-inc/packetfence/go/caddy/caddy"
	"github.com/inverse-inc/packetfence/go/caddy/caddy/caddyhttp/httpserver"
	"github.com/inverse-inc/packetfence/go/db"
	"github.com/inverse-inc/packetfence/go/log"
	"github.com/inverse-inc/packetfence/go/panichandler"
	"github.com/inverse-inc/packetfence/go/sharedutils"
	"github.com/jinzhu/gorm"
)

// Register the plugin in caddy
func init() {
	caddy.RegisterPlugin("pfpki", caddy.Plugin{
		ServerType: "http",
		Action:     setup,
	})
}

type (
	// Handler struct
	Handler struct {
		Next   httpserver.Handler
		router *mux.Router
		DB     *gorm.DB
		Ctx    context.Context
	}

	// GET Vars struct
	GetVars struct {
		Cursor int    `schema:"cursor" json:"cursor" default:"0"`
		Limit  int    `schema:"limit" json:"limit" default:"100"`
		Fields string `schema:"fields" json:"fields" default:"id"`
		Sort   string `schema:"sort" json:"sort" default:"id asc"`
		Query  Search `schema:"query" json:"query"`
	}

	// POST Vars struct
	PostVars struct {
		Cursor int      `schema:"cursor" json:"cursor" default:"0"`
		Limit  int      `schema:"limit" json:"limit" default:"100"`
		Fields []string `schema:"fields" json:"fields" default:"id"`
		Sort   []string `schema:"sort" json:"sort" default:"id asc"`
		Query  Search   `schema:"query" json:"query"`
	}

	// Search struct
	Search struct {
		Field  string      `schema:"field" json:"field,omitempty"`
		Op     string      `schema:"op" json:"op"`
		Value  interface{} `schema:"value" json:"value,omitempty"`
		Values []Search    `schema:"values" json:"values,omitempty"`
	}

	// Where struct
	Where struct {
		Query  string
		Values []interface{}
	}
)

// Setup the pfpki middleware
func setup(c *caddy.Controller) error {
	ctx := log.LoggerNewContext(context.Background())

	pfpki, err := buildPfpkiHandler(ctx)

	sharedutils.CheckError(err)

	httpserver.GetConfig(c).AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		pfpki.Next = next
		return pfpki
	})

	return nil
}

func buildPfpkiHandler(ctx context.Context) (Handler, error) {

	pfpki := Handler{}

	Database, err := gorm.Open("mysql", db.ReturnURI(ctx, "pf_pki"))
	sharedutils.CheckError(err)
	//pfpki.DB = Database
	pfpki.DB = Database.Debug()
	pfpki.Ctx = ctx

	// Default http timeout
	http.DefaultClient.Timeout = 10 * time.Second

	pfpki.router = mux.NewRouter()
	PFPki := &pfpki
	api := pfpki.router.PathPrefix("/api/v1").Subrouter()

	// CAs list
	api.Handle("/pki/cas", manageCA(PFPki)).Methods("GET")
	// Search CAs
	api.Handle("/pki/cas/search", manageCA(PFPki)).Methods("POST")
	// New CA
	api.Handle("/pki/ca", manageCA(PFPki)).Methods("POST")
	// Get CA by ID
	api.Handle("/pki/ca/{id}", manageCA(PFPki)).Methods("GET")

	// Profiles list
	api.Handle("/pki/profiles", manageProfile(PFPki)).Methods("GET")
	// Search Profiles
	api.Handle("/pki/profiles/search", manageProfile(PFPki)).Methods("POST")
	// New Profile
	api.Handle("/pki/profile", manageProfile(PFPki)).Methods("POST")
	// Get Profile by ID
	api.Handle("/pki/profile/{id}", manageProfile(PFPki)).Methods("GET")

	// Certificate list
	api.Handle("/pki/certs", manageCert(PFPki)).Methods("GET")
	// Search Certificates
	api.Handle("/pki/certs/search", manageCert(PFPki)).Methods("POST")
	// New Certificate
	api.Handle("/pki/cert", manageCert(PFPki)).Methods("POST")
	// Get Certificate by ID
	api.Handle("/pki/cert/{id}", manageCert(PFPki)).Methods("GET")
	// Download Certificate
	api.Handle("/pki/cert/{id}/download/{password}", manageCert(PFPki)).Methods("GET")
	// Get Certificate by email
	api.Handle("/pki/cert/{id}/email", manageCert(PFPki)).Methods("GET")
	// Revoke Certificate
	api.Handle("/pki/cert/{id}/{reason}", manageCert(PFPki)).Methods("DELETE")

	/*
		api.Handle("/pki/cert/getbycn/{cn}", manageCert(PFPki)).Methods("GET")
		// Get Certificate by id
		api.Handle("/pki/cert/getbyid/{id}", manageCert(PFPki)).Methods("GET")
		// Get Certificate by email
		api.Handle("/pki/certmgmt/{cn}", manageCert(PFPki)).Methods("GET")
		// Download Certificate
		api.Handle("/pki/certmgmt/{cn}/{password}", manageCert(PFPki)).Methods("GET")
		// Get Certificate by email
		api.Handle("/pki/certmgmt/getbyid/{id}", manageCert(PFPki)).Methods("GET")
		// Download Certificate
		api.Handle("/pki/certmgmt/getbyid/{id}/{password}", manageCert(PFPki)).Methods("GET")
		// Get Certificate by email
		api.Handle("/pki/certmgmt/getbycn/{cn}", manageCert(PFPki)).Methods("GET")
		// Download Certificate
		api.Handle("/pki/certmgmt/getbycn/{cn}/{password}", manageCert(PFPki)).Methods("GET")
		// Revoke Certificate
		api.Handle("/pki/cert/{cn}/{reason}", manageCert(PFPki)).Methods("DELETE")
	*/

	// OCSP responder
	api.Handle("/pki/ocsp", manageOcsp(PFPki)).Methods("GET", "POST")

	return pfpki, nil
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	ctx := r.Context()
	r = r.WithContext(ctx)

	defer panichandler.Http(ctx, w)

	routeMatch := mux.RouteMatch{}
	if h.router.Match(r, &routeMatch) {
		h.router.ServeHTTP(w, r)

		// TODO change me and wrap actions into something that handles server errors
		return 0, nil
	}
	return h.Next.ServeHTTP(w, r)
}