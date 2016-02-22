func (h *ConferenceApi) FilterPlayground(r *http.Request) (*ConferenceForms, error) {
	appCtx := appengine.NewContext(r)
	q := datastore.NewQuery("Conference")

	//simple filter usage:
	//q = q.Filter("City =", "Paris")

	//TODO
	//add 2 filters:
	//1: city equals to Chicago
	//2: topics equals "Medical Innovations"

	var conferences []Conference
	_, err := q.GetAll(appCtx, &conferences)
	if err != nil {
		return nil, err
	}

	forms := &ConferenceForms{
		Items: make([]ConferenceForm, 0, len(conferences)),
	}
	for v := range conferences {
		cf, _ := copyConferenceToForm(&conferences[v], conferences[v].Name, "")
		forms.Items = append(forms.Items, *cf)
	}
	return forms, nil
}