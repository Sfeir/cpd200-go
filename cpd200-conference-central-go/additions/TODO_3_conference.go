func (h *ConferenceApi) QueryConferences(r *http.Request, cqf *ConferenceQueryForms) (*ConferenceForms, error) {
	//Query for conferences.
	appCtx := appengine.NewContext(r)
	q := datastore.NewQuery("Conference")
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
		[]string{WEB_CLIENT_ID}, []string{WEB_CLIENT_ID})
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