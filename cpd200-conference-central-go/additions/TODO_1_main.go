func SendConfirmationEmailHandler(w http.ResponseWriter, r *http.Request) {
	//Send email confirming Conference creation.
	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}
	header := r.Header.Get("X-AppEngine-QueueName")
	if header == "" {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("attempt to access task handler directly, missing custom App Engine header"))
		return
	}
	appCtx := appengine.NewContext(r)
	appId := appengine.AppID(appCtx)
	msg := &mail.Message{
		Sender: "noreply@" + appId + ".appspotmail.com",
		To:	[]string{r.PostFormValue("email")},
		Subject: "You created a new Conference!",
		Body: "Hi, you have created a following conference:\r\n\r\n" +
			r.PostFormValue("conferenceInfo"),
	}
	err := mail.Send(appCtx, msg)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}