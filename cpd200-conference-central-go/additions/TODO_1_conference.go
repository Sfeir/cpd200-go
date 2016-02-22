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