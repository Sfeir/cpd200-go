// Copyright 2015 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package admin

import (
	"html/template"
	"net/http"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/user"
)

// [START alert_struct]
type Alert struct {
	Author  string    `datastore:"author"`
	Content string    `datastore:"content"`
	Date    time.Time `datastore:"date"`
}

// [END alert_struct]

func init() {
	http.HandleFunc("/", root)
	http.HandleFunc("/add", add)
}

// alertKey returns the key used for all alert entries.
func alertKey(c appengine.Context) *datastore.Key {
	// The string "default_feed" here could be varied to have multiple feeds.
	return datastore.NewKey(c, "Alert", "default_feed", 0, nil)
}

// [START func_root]
func root(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	// Ancestor queries, as shown here, are strongly consistent with Cloud Datastore.
	// Queries that span entity groups are eventually
	// consistent. If we omitted the .Ancestor from this query there would be
	// a slight chance that Alert that had just been written would not
	// show up in a query.
	// [START query]
	q := datastore.NewQuery("Alert")
	// q := datastore.NewQuery("Alert").Ancestor(alertKey(c)).Order("-date").Limit(10)
	// [END query]
	// [START getall]
	alerts := make([]Alert, 0, 10)
	if _, err := q.GetAll(c, &alerts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// [END getall]
	if err := feedTemplate.Execute(w, alerts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// [END func_root]

var feedTemplate = template.Must(template.ParseFiles("index.html"))

// [START func_alert]
func add(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	a := Alert{
		Content: r.FormValue("content"),
		Date:    time.Now(),
	}
	if u := user.Current(c); u != nil {
		a.Author = u.String()
	}
	// We set the same parent key on every Alert entity to ensure each
	// Alert is in the same entity group. Queries across the single entity
	// group will be consistent. However, the write rate to a single entity group
	// should be limited to ~1/second.
	key := datastore.NewIncompleteKey(c, "Alert", alertKey(c))
	_, err := datastore.Put(c, key, &a)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

// [END func_alert]
