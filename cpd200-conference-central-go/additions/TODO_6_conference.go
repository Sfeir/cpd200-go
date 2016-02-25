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