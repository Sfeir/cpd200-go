package main

/*
conference.go -- server-side Go App Engine API;
    uses Google Cloud Endpoints

*/


import (
	"log"
	applog "google.golang.org/appengine/log"
	"github.com/GoogleCloudPlatform/go-endpoints/endpoints"
	"net/http"
	"google.golang.org/appengine"
)

type ConferenceApi struct {
}

func copyProfileToForm(r *http.Request, prof *Profile) (*ProfileForm, error) {
	//Copy relevant fields from Profile to ProfileForm.
	pf := &ProfileForm{
			DisplayName: prof.DisplayName,
			MainEmail: prof.MainEmail,
			TeeShirtSize: StringEnumToTeeShirtSize(prof.TeeShirtSize),
	}
	appCtx := appengine.NewContext(r)
	applog.Debugf(appCtx, "Did run copyProfileToForm()")
	return pf, nil
}

func getProfileFromUser(r *http.Request) (*Profile, error) {
	//Return user Profile from datastore, creating new one if non-existent.
	//TODO
	//make sure user is authed
	c := endpoints.NewContext(r)
	user, err := endpoints.CurrentUser(c, []string{endpoints.EmailScope},
		[]string{WEB_CLIENT_ID, endpoints.APIExplorerClientID}, []string{WEB_CLIENT_ID, endpoints.APIExplorerClientID})
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, endpoints.UnauthorizedError
	}
	var profile *Profile
	if profile == nil {
		profile = &Profile{
			DisplayName: user.String(),
			MainEmail: user.Email,
			TeeShirtSize: TeeShirtSizeToStringEnum(NOT_SPECIFIED),
		}
		//TODO
	}
	return profile, nil
}

func doProfile(r *http.Request, saveRequest *ProfileMiniForm) (*ProfileForm, error) {
	//Get user Profile and return to user, possibly updating it first.
	//get user Profile
	prof, err := getProfileFromUser(r)
	if err != nil {
		return nil, err
	}
	
	//if saveProfile(), process user-modifyable fields
	if saveRequest != nil {
		prof.TeeShirtSize = TeeShirtSizeToStringEnum(saveRequest.TeeShirtSize)
		prof.DisplayName = saveRequest.DisplayName
	}
	
	//return ProfileForm
	return copyProfileToForm(r, prof)
}

func (h *ConferenceApi) GetProfile(r *http.Request) (*ProfileForm, error) {
	//Return user profile.
	return doProfile(r, nil)
}

func (h *ConferenceApi) SaveProfile(r *http.Request, pf *ProfileMiniForm) (*ProfileForm, error) {
	//Update & return user profile.
	return doProfile(r, pf)
}

func init() {
	//Conference API v0.1
	conference := &ConferenceApi{}
	//registers API
	api, err := endpoints.RegisterService(conference, "conference", "v1", "Conference API", true)
	if err != nil {
		log.Fatalf("Register service: %v", err)
	}
	
	register := func(orig, name, method, path, desc string) {
		m := api.MethodByName(orig)
		if m == nil {
			log.Fatalf("Missing method %s", orig)
		}
		i := m.Info()
		i.Name, i.HTTPMethod, i.Path, i.Desc = name, method, path, desc
		i.Scopes = []string{endpoints.EmailScope}
		i.ClientIds = []string{WEB_CLIENT_ID, endpoints.APIExplorerClientID}
	}

	register("GetProfile", "getProfile", "GET", "profile", "Get profile")
	register("SaveProfile", "saveProfile", "POST", "profile", "Save profile")
	endpoints.HandleHTTP()
}
