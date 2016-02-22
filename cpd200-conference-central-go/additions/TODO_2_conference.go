type Object struct {
	Value interface{}
}

var DEFAULTS = map[string]Object{
	"city": Object{Value:"Default City"},
	"maxAttendees": Object{Value:0},
	"seatsAvailable": Object{Value:0},
	"topics": Object{Value:[2]string{"Default", "Topic"}},
}

func copyConferenceToForm(conf *Conference, key datastore.Key, displayName string) (*ConferenceForm, error) {
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
		WebsafeKey: html.EscapeString(key.String()),
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
		[]string{WEB_CLIENT_ID}, []string{WEB_CLIENT_ID})
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

func (h *ConferenceApi) CreateConference(r *http.Request, cf *ConferenceForm) (*ConferenceForm, error) {
	//Create new conference.
	return createConferenceObject(r, cf)
}