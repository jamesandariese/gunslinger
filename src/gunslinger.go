package gunslinger

import (
	"fmt"
	"net/http"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/plus/v1"

	"encoding/base64"
	"github.com/jamesandariese/uuid4"
	"strings"

	"google.golang.org/appengine/log"
)

type stringWrapper struct {
	S string
}

func mustConfigFromJSON(configJson string, scopes ...string) *oauth2.Config {
	config, err := google.ConfigFromJSON([]byte(configJson), scopes...)
	if err != nil {
		panic(err)
	}
	return config
}

// the HTTP context
// this variable must be set at each handler entry point
var ctx context.Context

var config = mustConfigFromJSON(configJson, gmail.MailGoogleComScope, plus.UserinfoEmailScope)

func dsMustPut(typ, skey string, src interface{}) {
	key := datastore.NewKey(ctx, typ, skey, 0, nil)
	if _, err := datastore.Put(ctx, key, src); err != nil {
		panic(err)
	}
}

func dsMustDelete(typ, skey string) {
	key := datastore.NewKey(ctx, typ, skey, 0, nil)
	if err := datastore.Delete(ctx, key); err != nil {
		panic(err)
	}
}

func dsMustGet(typ, skey string, dst interface{}) {
	key := datastore.NewKey(ctx, typ, skey, 0, nil)
	if err := datastore.Get(ctx, key, dst); err != nil {
		panic(err)
	}
}

func dsMayGet(typ, skey string, dst interface{}) (found bool) {
	key := datastore.NewKey(ctx, typ, skey, 0, nil)
	if err := datastore.Get(ctx, key, dst); err != nil {
		if err == datastore.ErrNoSuchEntity {
			return false
		}
		panic(err)
	}
	return true
}

func oauthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx = appengine.NewContext(r)

	state := r.FormValue("state")
	dsMustDelete("OAuthCode", state)

	code := r.FormValue("code")

	tok, err := config.Exchange(ctx, code)
	if err != nil {
		panic(err)
	}

	tokenSource := config.TokenSource(ctx, tok)
	token, err := tokenSource.Token()
	if err != nil {
		panic(err)
	}

	client := config.Client(ctx, token)
	if err != nil {
		panic(err)
	}

	plusSvc, err := plus.New(client)
	if err != nil {
		panic(err)
	}

	peopleSvc := plus.NewPeopleService(plusSvc)
	me, err := peopleSvc.Get("me").Do()
	if err != nil {
		panic(err)
	}

	email := ""
	foundAccountEmail := false
	for _, e := range me.Emails {
		if e.Type != "account" {
			continue
		}
		email = e.Value
		foundAccountEmail = true
	}

	if !foundAccountEmail {
		panic("poopers.  there was no account email address?  this is actually impossible")
	}

	webhook := generateWebhookValue()
	dsMustPut("WebhookToEmail", webhook, &stringWrapper{email})
	dsMustPut("WebhookToToken", webhook, tok)

	http.Redirect(w, r, "/webhook/"+webhook, http.StatusFound)
}

// get a new webhook for the user
// saves it to the datastore
func generateWebhookValue() string {
	u1, u2 := uuid4.NewUUID(), uuid4.NewUUID()
	x := u1.HexString() + u2.HexString()
	var tmp stringWrapper
	for dsMayGet("WebhookToToken", x, &tmp) {
		// let's make a new one as long as the one we generated exists
		// this is *exceedingly unlikely* but you know.  whatever.
		u1, u2 = uuid4.NewUUID(), uuid4.NewUUID()
		x = u1.HexString() + u2.HexString()
	}
	return x
}

// get the Token object for the webhook, refresh it, and save
// the refreshed token
func getTokenForWebhook(webhook string) *oauth2.Token {
	token := new(oauth2.Token)
	dsMustGet("WebhookToToken", webhook, token)
	tokenSource := config.TokenSource(ctx, token)
	token, err := tokenSource.Token()
	if err != nil {
		panic(err)
	}
	dsMustPut("WebhookToToken", webhook, token)
	return token
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	ctx = appengine.NewContext(r)

	if len(r.URL.Path) < 9 {
		panic("invalid webhook")
	}
	webhook := r.URL.Path[9:]

	err := r.ParseForm()
	if err != nil {
		panic(err)
	}

	log.Debugf(ctx, "%v\n", r.Form["body-mime"])

	var email stringWrapper
	dsMustGet("WebhookToEmail", webhook, &email)
	token := getTokenForWebhook(webhook)
	fmt.Fprintf(w, "Welcome %s\n", email.S)

	client := config.Client(appengine.NewContext(r), token)

	srv, err := gmail.New(client)
	if err != nil {
		panic(err)
	}

	switch r.Method {
	case "POST", "PUT":
		msgsrv := gmail.NewUsersMessagesService(srv)
		msg := &gmail.Message{
			LabelIds: []string{"INBOX", "UNREAD"},
			Raw:      base64.RawURLEncoding.EncodeToString([]byte(strings.Join(r.Form["body-mime"], "\n"))),
		}
		msg, err = msgsrv.Import("me", msg).Do()
		fmt.Fprintf(w, "%#v %#v\n", msg, err)
	case "GET":
	}
}

func init() {
	http.HandleFunc("/oauth2callback", oauthCallbackHandler)
	http.HandleFunc("/", handler)
	http.HandleFunc("/webhook/", webhookHandler)
}

type oauthCode struct {
	filler string
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx = appengine.NewContext(r)
	randomString := generateWebhookValue()

	dsMustPut("OAuthCode", randomString, &oauthCode{"filler"})
	url := config.AuthCodeURL(randomString, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	http.Redirect(w, r, url, http.StatusFound)
}
