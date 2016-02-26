var MEMCACHE_ANNOUNCEMENTS_KEY = "RECENT_ANNOUNCEMENTS​"

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