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
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"time"
	"html"
	"strconv"
	"google.golang.org/appengine/memcache"
)

type ConferenceApi struct {
}

var MEMCACHE_ANNOUNCEMENTS_KEY = "RECENT_ANNOUNCEMENTS​"

type Object struct {
	Value interface{}
}

var DEFAULTS = map[string]Object{
	"city": Object{Value:"Default City"},
	"maxAttendees": Object{Value:0},
	"seatsAvailable": Object{Value:0},
	"topics": Object{Value:[2]string{"Default", "Topic"}},
}

var OPERATORS = map[string]string{
	"EQ": "=",
	"GT": ">",
	"GTEQ": ">=",
	"LT": "<",
	"LTEQ": "<=",
	"NE": "!=",
}

var FIELDS = map[string]string{
	"CITY": "City",
	"TOPIC": "Topics",
	"MONTH": "Month",
	"MAX_ATTENDEES": "MaxAttendees",
}

func copyConferenceToForm(conf *Conference, keyStr string, displayName string) (*ConferenceForm, error) {
	//Copy relevant fields from Conference to ConferenceForm.
	cf := &ConferenceForm{
		Name: conf.Name,
		Description: conf.Description,
		OrganizerUserId: conf.OrganizerUserId,
		Topics: conf.Topics,
		City: conf.City,
		StartDate: conf.StartDate.String(),
		Month: conf.Month,
		MaxAttendees: conf.MaxAttendees,
		SeatsAvailable: conf.SeatsAvailable,
		EndDate: conf.EndDate.String(),
		WebsafeKey: html.EscapeString(keyStr),
	}
	if displayName != "" {
		cf.OrganizerDisplayName = displayName
	}
	return cf, nil
}

func createConferenceObject(r *http.Request, cf *ConferenceForm) (*ConferenceForm, error) {
	//Create or update Conference object, returning ConferenceForm.
	//preload necessary data items
	c := endpoints.NewContext(r)
	user, err := endpoints.CurrentUser(c, []string{endpoints.EmailScope},
		[]string{WEB_CLIENT_ID, endpoints.APIExplorerClientID}, []string{WEB_CLIENT_ID, endpoints.APIExplorerClientID})
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, endpoints.UnauthorizedError
	}
	userId := getUserId(user, "")
	
	if cf.Name == "" {
		return nil, endpoints.BadRequestError
	}
	
	cf.WebsafeKey = ""
	cf.OrganizerDisplayName = ""

	//add default values for those missing
	if cf.City == "" {
		cf.City = DEFAULTS["city"].Value.(string)
	}
	if cf.MaxAttendees == 0 {
		cf.MaxAttendees = DEFAULTS["maxAttendees"].Value.(int)
	}
	if cf.SeatsAvailable == 0 {
		cf.SeatsAvailable = DEFAULTS["seatsAvailable"].Value.(int)
	}
	if cf.Topics == nil {
		cf.Topics = DEFAULTS["topics"].Value.([]string)
	}

	//convert dates from strings to Date objects; set month based on start_date
	var startDate time.Time
	var endDate time.Time
	if cf.StartDate != "" {
		startDate, _ = time.Parse(time.RFC3339, cf.StartDate)
		cf.Month = int(startDate.Month())
	} else {
		cf.Month = 0
	}
	if cf.EndDate != "" {
		endDate, _ = time.Parse(time.RFC3339, cf.EndDate)
	}

	//set seatsAvailable to be same as maxAttendees on creation
	//both for data model & outbound Message
	if cf.MaxAttendees > 0 {
		cf.SeatsAvailable = cf.MaxAttendees
	}

	//make Profile Key from user ID
	appCtx := appengine.NewContext(r)
	parentKey := datastore.NewKey(appCtx, "Profile", userId, 0, nil)
	//allocate new Conference ID with Profile key as parent
	_, high, err := datastore.AllocateIDs(appCtx, "Conference", parentKey, 1)
	if err != nil {
		return nil, err
	}
	//make Conference key from ID
	confKey := datastore.NewKey(appCtx, "Conference", "", high, parentKey)
	cf.OrganizerUserId = userId

	//create Conference & return (modified) ConferenceForm
	conf := &Conference{
		Name: cf.Name,
		Description: cf.Description,
		OrganizerUserId: cf.OrganizerUserId,
		Topics: cf.Topics,
		City: cf.City,
		StartDate: startDate,
		Month: cf.Month,
		EndDate: endDate,
		MaxAttendees: cf.MaxAttendees,
		SeatsAvailable: cf.SeatsAvailable,
	}
	_, err = datastore.Put(appCtx, confKey, conf)
	if err != nil {
		return nil, err
	}

	return cf, nil
}

func getQuery(appCtx context.Context, cqf *ConferenceQueryForms) (*datastore.Query, error) {
	//Return formatted query from the submitted filters.
	q := datastore.NewQuery("Conference")
	inequalityFilter, filters, err := formatFilters(cqf.Filters)
	if err != nil {
		return nil, err
	}

	//If exists, sort on inequality filter first
	if inequalityFilter == "" {
		q = q.Order("Name")
	} else {
		q = q.Order(inequalityFilter)
		q = q.Order("Name")
	}
	
	for v := range filters {
		filtr := filters[v]
		
		if filtr.Field == "Month" || filtr.Field == "MaxAttendees" {
			val, err := strconv.Atoi(filtr.Value)
			if err != nil {
				return nil, err
			}
			q = q.Filter(filtr.Field + filtr.Operator, val)
		} else {
			q = q.Filter(filtr.Field + filtr.Operator, filtr.Value)
		}
	}
	
	return q, nil
}

func formatFilters(filters []ConferenceQueryForm) (string, []ConferenceQueryForm, error) {
	//Parse, check validity and format user supplied filters.
	formattedFilters := make([]ConferenceQueryForm, 0, len(filters))
	inequalityField := ""
	
	for v := range filters {
		filtr := filters[v]
				
		if val, ok := FIELDS[filtr.Field]; ok {
			filtr.Field = val
		} else {
			return "", nil, endpoints.BadRequestError
		}
		if val, ok := OPERATORS[filtr.Operator]; ok {
			filtr.Operator = val
		} else {
			return "", nil, endpoints.BadRequestError
		}
		
		//Every operation except "=" is an inequality
		if filtr.Operator != "=" {
			//check if inequality operation has been used in previous filters
			//disallow the filter if inequality was performed on a different field before
			//track the field on which the inequality operation is performed
			if inequalityField != "" && inequalityField != filtr.Field {
				return "", nil, endpoints.BadRequestError
			} else {
				inequalityField = filtr.Field
			}
		}
		
		formattedFilters = append(formattedFilters, filtr)
	}

	return inequalityField, formattedFilters, nil
}

func (h *ConferenceApi) QueryConferences(r *http.Request, cqf *ConferenceQueryForms) (*ConferenceForms, error) {
	//Query for conferences.
	appCtx := appengine.NewContext(r)
	q, err := getQuery(appCtx, cqf)
	if err != nil {
		return nil, err
	}
	var conferences []Conference
	keys, err := q.GetAll(appCtx, &conferences)
	if err != nil {
		return nil, err
	}

	//return individual ConferenceForm object per Conference
	forms := &ConferenceForms{
		Items: make([]ConferenceForm, 0, len(conferences)),
	}
	for v := range conferences {
		cf, _ := copyConferenceToForm(&conferences[v], keys[v].Encode(), "")
		forms.Items = append(forms.Items, *cf)
	}
	return forms, nil
}

func (h *ConferenceApi) GetConferencesCreated(r *http.Request) (*ConferenceForms, error) {
	//Return conferences created by user.
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

	//make profile key
	userId := getUserId(user, "")
	appCtx := appengine.NewContext(r)
	parentKey := datastore.NewKey(appCtx, "Profile", userId, 0, nil)
	//create ancestor query for this user
	q := datastore.NewQuery("Conference").Ancestor(parentKey)
	var conferences []Conference
	keys, err := q.GetAll(appCtx, &conferences)
	if err != nil {
		return nil, err
	}
	//get the user profile and display name
	var profile Profile
	err = datastore.Get(appCtx, parentKey, &profile)
	if err != nil && err != datastore.ErrNoSuchEntity {
		return nil, err
	}
	displayName := profile.DisplayName
	//return set of ConferenceForm objects per Conference
	forms := &ConferenceForms{
		Items: make([]ConferenceForm, 0, len(conferences)),
	}
	for v := range conferences {
		cf, _ := copyConferenceToForm(&conferences[v], keys[v].Encode(), displayName)
		forms.Items = append(forms.Items, *cf)
	}
	return forms, nil
}

func (h *ConferenceApi) CreateConference(r *http.Request, cf *ConferenceForm) (*ConferenceForm, error) {
	//Create new conference.
	return createConferenceObject(r, cf)
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

func getProfileFromUser(r *http.Request) (*Profile, *datastore.Key, error) {
	//Return user Profile from datastore, creating new one if non-existent.
	//TODO
	//make sure user is authed
	c := endpoints.NewContext(r)
	user, err := endpoints.CurrentUser(c, []string{endpoints.EmailScope},
		[]string{WEB_CLIENT_ID, endpoints.APIExplorerClientID}, []string{WEB_CLIENT_ID, endpoints.APIExplorerClientID})
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, endpoints.UnauthorizedError
	}
	//get Profile from datastore
	userId := getUserId(user, "")
	appCtx := appengine.NewContext(r)
	key := datastore.NewKey(appCtx, "Profile", userId, 0, nil)
	var profile Profile
	err = datastore.Get(appCtx, key, &profile)
	if err != nil && err != datastore.ErrNoSuchEntity {
		return nil, nil, err
	}
	if err == datastore.ErrNoSuchEntity {
		profile = Profile{
			DisplayName: user.String(),
			MainEmail: user.Email,
			TeeShirtSize: TeeShirtSizeToStringEnum(NOT_SPECIFIED),
		}
		_, err := datastore.Put(appCtx, key, &profile)
		if err != nil {
			return nil, nil, err
		}
	}
	return &profile, key, nil
}

func doProfile(r *http.Request, saveRequest *ProfileMiniForm) (*ProfileForm, error) {
	//Get user Profile and return to user, possibly updating it first.
	//get user Profile
	prof, key, err := getProfileFromUser(r)
	if err != nil {
		return nil, err
	}
	
	//if saveProfile(), process user-modifyable fields
	if saveRequest != nil {
		prof.TeeShirtSize = TeeShirtSizeToStringEnum(saveRequest.TeeShirtSize)
		prof.DisplayName = saveRequest.DisplayName
		appCtx := appengine.NewContext(r)
		_, err := datastore.Put(appCtx, key, prof)
		if err != nil {
			return nil, err
		}
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

func cacheAnnouncement(r *http.Request) (string, error) {
	//Create Announcement & assign to memcache; used by memcache cron job & putAnnouncement().
	appCtx := appengine.NewContext(r)
	q := datastore.NewQuery("Conference").
		Filter("SeatsAvailable<=", 5).
		Filter("SeatsAvailable>", 0).
		Project("Name")
	var confs []Conference
	_, err := q.GetAll(appCtx, &confs)
	if err != nil {
		return "", err
	}
	
	var announcement string
	if len(confs) > 0 {
		//If there are almost sold out conferences,
		//format announcement and set it in memcache
		announcement = "Last chance to attend! The following conferences are nearly sold out: "
		for v := range confs {
			announcement += confs[v].Name + ", "
		}
		item := &memcache.Item{
		    Key:   MEMCACHE_ANNOUNCEMENTS_KEY,
		    Value: []byte(announcement),
		}
		memcache.Set(appCtx, item)
	} else {
		//If there are no sold out conferences,
		//delete the memcache announcements entry
		announcement = ""
		memcache.Delete(appCtx, MEMCACHE_ANNOUNCEMENTS_KEY)
	}
	
	return announcement, nil
}

func (h *ConferenceApi) GetAnnouncement(r *http.Request) (*StringMessage, error) {
	//Return Announcement from memcache.
	appCtx := appengine.NewContext(r)
	found, err := memcache.Get(appCtx, MEMCACHE_ANNOUNCEMENTS_KEY)
	if err != nil && err != memcache.ErrCacheMiss {
		return nil, err
	}
	var data string
	if err == memcache.ErrCacheMiss {
		data = ""
	} else {
		data = string(found.Value)
	}
	return &StringMessage{Data: data}, nil
}

func conferenceRegistration(websafeConferenceKey string, r *http.Request, reg bool) (*BooleanMessage, error) {
	//Register or unregister user for selected conference.
	var retval bool
	appCtx := appengine.NewContext(r)
	err := datastore.RunInTransaction(appCtx, func(appCtx context.Context) error {
		prof, profKey, err := getProfileFromUser(r) //get user Profile
		if err != nil {
			return err
		}
		
		//check if conf exists given websafeConfKey
		//get conference; check that it exists
		confKey, err := datastore.DecodeKey(websafeConferenceKey)
		if err != nil {
			return err
		}
		var conf Conference
		err = datastore.Get(appCtx, confKey, &conf)
		if err != nil && err != datastore.ErrNoSuchEntity {
			return err
		}
		if err == datastore.ErrNoSuchEntity {
			return endpoints.NotFoundError
		}
	
		alreadyRegistered := -1
		for i, k := range prof.ConferenceKeysToAttend {
			if k == websafeConferenceKey {
				alreadyRegistered = i
				break
			}
		}
		
		//register
		if reg {
			//check if user already registered otherwise add
			if alreadyRegistered > -1 {
				return endpoints.NewConflictError("You have already registered for this conference")
			}
			
			//check if seats avail
			if conf.SeatsAvailable <= 0 {
				return endpoints.NewConflictError("There are no seats available.")
			}
			
			//register user, take away one seat
			prof.ConferenceKeysToAttend = append(prof.ConferenceKeysToAttend, websafeConferenceKey)
			conf.SeatsAvailable -= 1
			retval = true
		} else {	//unregister
			//check if user already registered
			if alreadyRegistered > -1 {
				//unregister user, add back one seat
				i := alreadyRegistered
				prof.ConferenceKeysToAttend = append(prof.ConferenceKeysToAttend[:i], prof.ConferenceKeysToAttend[i+1:]...)
				conf.SeatsAvailable += 1
				retval = true
			} else {
				retval = false
			}
		}
		
		//write things back to the datastore & return
		_, err = datastore.Put(appCtx, profKey, prof)
		if err != nil {
			return err
		}
		_, err = datastore.Put(appCtx, confKey, &conf)
		if err != nil {
			return err
		}
		return err
	}, nil)
	if err != nil {
		return nil, err
	}
	return &BooleanMessage{Data:retval}, nil
}

func (h *ConferenceApi) GetConferencesToAttend(r *http.Request) (*ConferenceForms, error) {
	//Get list of conferences that user has registered for.
	prof, _, err := getProfileFromUser(r) //get user Profile
	if err != nil {
		return nil, err
	}
	confKeys := make([]*datastore.Key, 0, len(prof.ConferenceKeysToAttend))
	for v := range prof.ConferenceKeysToAttend {
		key, err := datastore.DecodeKey(prof.ConferenceKeysToAttend[v])
		if err != nil {
			return nil, err
		}
		confKeys = append(confKeys, key)
	}
	appCtx := appengine.NewContext(r)
	conferences := make([]Conference, len(confKeys))
	err = datastore.GetMulti(appCtx, confKeys, conferences)
	if err != nil {
		return nil, err
	}
	
	//get organizers
	organisers := make([]*datastore.Key, 0, len(conferences))
	for v := range conferences {
		key := datastore.NewKey(appCtx,"Profile", conferences[v].OrganizerUserId, 0, nil)
		organisers = append(organisers, key)
	}
	profiles := make([]Profile, len(organisers))
	err = datastore.GetMulti(appCtx, organisers, profiles)
	if err != nil {
		return nil, err
	}
	
	//put display names in a dict for easier fetching
	names := make(map[string]string)
	for v := range profiles {
		names[organisers[v].StringID()] = profiles[v].DisplayName
	}
	
	//return set of ConferenceForm objects per Conference
	forms := &ConferenceForms{
		Items: make([]ConferenceForm, 0, len(conferences)),
	}
	for v := range conferences {
		cf, _ := copyConferenceToForm(&conferences[v], confKeys[v].Encode(), names[conferences[v].OrganizerUserId])
		forms.Items = append(forms.Items, *cf)
	}
	return forms, nil
}

type ConfRequest struct {
	WebsafeConferenceKey string	`json:"websafeConferenceKey"`
}

func (h *ConferenceApi) RegisterForConference(r *http.Request, cr *ConfRequest) (*BooleanMessage, error) {
	//Register user for selected conference.
	return conferenceRegistration(cr.WebsafeConferenceKey, r, true)
}

func (h *ConferenceApi) GetConference(r *http.Request, cr *ConfRequest) (*ConferenceForm, error) {
	//Return requested conference (by websafeConferenceKey).
	//get Conference object from request; bail if not found
	key, err := datastore.DecodeKey(cr.WebsafeConferenceKey)
	if err != nil {
		return nil, err
	}
	var conf Conference
	appCtx := appengine.NewContext(r)
	err = datastore.Get(appCtx, key, &conf)
	if err != nil && err != datastore.ErrNoSuchEntity {
		return nil, err
	}
	if err == datastore.ErrNoSuchEntity {
		return nil, endpoints.NotFoundError
	}
	parentKey := key.Parent()
	var prof Profile
	err = datastore.Get(appCtx, parentKey, &prof)
	if err != nil && err != datastore.ErrNoSuchEntity {
		return nil, err
	}
	//return ConferenceForm
	displayName := prof.DisplayName
	return copyConferenceToForm(&conf, key.Encode(), displayName)
}

func doAlert(r *http.Request) (*LatestAlert, error) {
	//Get latest alert and return to user
	//retrieve latest alert from datastore
	appCtx := appengine.NewContext(r)
	q := datastore.NewQuery("Alert").Order("-date")
	var alerts []Alert
	_, err := q.GetAll(appCtx, &alerts)
	if err != nil {
		return nil, err
	}
	la := &LatestAlert{}
	if len(alerts) > 0 {
		la.Content = alerts[0].Content
	}
	return la, nil
}

func (h *ConferenceApi) GetAlert(r *http.Request) (*LatestAlert, error) {
	//Return latest alert.
	return doAlert(r)
}

func (h *ConferenceApi) FilterPlayground(r *http.Request) (*ConferenceForms, error) {
	appCtx := appengine.NewContext(r)
	q := datastore.NewQuery("Conference")

	//simple filter usage:
	//q = q.Filter("City =", "Paris")

	//TODO
	//add 2 filters:
	//1: city equals to Chicago
	//2: topics equals "Medical Innovations"
	q = q.Filter("City=", "Chicago")
	q = q.Filter("Topics=", "Medical Innovations")

	var conferences []Conference
	keys, err := q.GetAll(appCtx, &conferences)
	if err != nil {
		return nil, err
	}

	forms := &ConferenceForms{
		Items: make([]ConferenceForm, 0, len(conferences)),
	}
	for v := range conferences {
		cf, _ := copyConferenceToForm(&conferences[v], keys[v].Encode(), "")
		forms.Items = append(forms.Items, *cf)
	}
	return forms, nil
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
	register("CreateConference", "createConference", "POST", "conference", "Create conference")
	register("QueryConferences", "queryConferences", "POST", "queryConferences", "Query conferences")
	register("GetConferencesCreated", "getConferencesCreated", "POST", "getConferencesCreated", "Get conferences created")
	register("FilterPlayground", "filterPlayground", "GET", "filterPlayground", "Filter playground")
	register("RegisterForConference", "registerForConference", "POST", "conference/{websafeConferenceKey}", "Register for conference")
	register("GetConference", "getConference", "GET", "conference/{websafeConferenceKey}", "Get conference")
	register("GetConferencesToAttend", "getConferencesToAttend", "GET", "conferences/attending", "Get conferences to attend")
	register("GetAlert", "getAlert", "GET", "alert", "Get alert")
	register("GetAnnouncement", "getAnnouncement", "GET", "conference/announcement/get", "Get announcement")
	endpoints.HandleHTTP()
}
