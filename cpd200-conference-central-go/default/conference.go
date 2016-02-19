package main

/*
conference.go -- server-side Go App Engine API;
    uses Google Cloud Endpoints

*/


import (
	"log"
	"github.com/GoogleCloudPlatform/go-endpoints/endpoints"
	"net/http"
)

type ConferenceApi struct {
}

func copyProfileToForm(prof *Profile) (*ProfileForm, error) {
	//Copy relevant fields from Profile to ProfileForm.
	pf := &ProfileForm{
			DisplayName: prof.DisplayName,
			MainEmail: prof.MainEmail,
			TeeShirtSize: StringEnumToTeeShirtSize(prof.TeeShirtSize),
	}
	return pf, nil
}

func getProfileFromUser(r *http.Request) (*Profile, error) {
	//Return user Profile from datastore, creating new one if non-existent.
	//TODO
	//make sure user is authed
	//c := endpoints.NewContext(r)
	//user, err := endpoints.CurrentUser(c, []string{endpoints.EmailScope},
	//	[]string{WEB_CLIENT_ID}, []string{WEB_CLIENT_ID})
	//if err != nil {
	//	return nil, err
	//}
	//if user == nil {
	//	return nil, endpoints.UnauthorizedError
	//}
	var profile *Profile
	if profile == nil {
		profile = &Profile{
			DisplayName: "Test",
			MainEmail: "",
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
	return copyProfileToForm(prof)
}

func (h *ConferenceApi) GetProfile(r *http.Request) (*ProfileForm, error) {
	//Return user profile.
	return doProfile(r, nil)
}

func (h *ConferenceApi) SaveProfile(r *http.Request) (*ProfileForm, error) {
	//Update & return user profile.
	return doProfile(r, nil)
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
	}

	register("GetProfile", "getProfile", "GET", "profile", "Get profile")
	register("SaveProfile", "saveProfile", "POST", "profile", "Save profile")
	endpoints.HandleHTTP()
}
