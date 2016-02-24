	//create Conference, send email to organizer confirming
	//creation of Conference & return (modified) ConferenceForm
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
	js, _ := json.Marshal(cf);
	task := taskqueue.NewPOSTTask("/tasks/send_confirmation_email", url.Values{
	    "email": {user.Email},
	    "conferenceInfo": {string(js)},
	})
	taskqueue.Add(appCtx, task, "")