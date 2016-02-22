package main

import (
	appuser "google.golang.org/appengine/user"
)

func getUserId(user *appuser.User, idType string) string {
	if idType == "" || idType == "email" {
		return user.Email
	}
	
	if idType == "oauth" {
		
	}
	
	if idType == "custom" {
		
	}
	
	return ""
}
