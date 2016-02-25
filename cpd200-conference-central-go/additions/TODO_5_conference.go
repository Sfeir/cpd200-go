var OPERATORS = map[string]string{
	"EQ": "=",
	"GT": ">",
	"GTEQ": ">=",
	"LT": "<",
	"LTEQ": "<=",
	"NE": "!=",
}

var FIELDS = map[string]string{
	"CITY": "City",
	"TOPIC": "Topics",
	"MONTH": "Month",
	"MAX_ATTENDEES": "MaxAttendees",
}


func getQuery(appCtx context.Context, cqf *ConferenceQueryForms) (*datastore.Query, error) {
	//Return formatted query from the submitted filters.
	q := datastore.NewQuery("Conference")
	inequalityFilter, filters, err := formatFilters(cqf.Filters)
	if err != nil {
		return nil, err
	}

	//If exists, sort on inequality filter first
	if inequalityFilter == "" {
		q = q.Order("Name")
	} else {
		q = q.Order(inequalityFilter)
		q = q.Order("Name")
	}
	
	for v := range filters {
		filtr := filters[v]
		
		if filtr.Field == "Month" || filtr.Field == "MaxAttendees" {
			val, err := strconv.Atoi(filtr.Value)
			if err != nil {
				return nil, err
			}
			q = q.Filter(filtr.Field + filtr.Operator, val)
		} else {
			q = q.Filter(filtr.Field + filtr.Operator, filtr.Value)
		}
	}
	
	return q, nil
}

func formatFilters(filters []ConferenceQueryForm) (string, []ConferenceQueryForm, error) {
	//Parse, check validity and format user supplied filters.
	formattedFilters := make([]ConferenceQueryForm, 0, len(filters))
	inequalityField := ""
	
	for v := range filters {
		filtr := filters[v]
				
		if val, ok := FIELDS[filtr.Field]; ok {
			filtr.Field = val
		} else {
			return "", nil, endpoints.BadRequestError
		}
		if val, ok := OPERATORS[filtr.Operator]; ok {
			filtr.Operator = val
		} else {
			return "", nil, endpoints.BadRequestError
		}
		
		//Every operation except "=" is an inequality
		if filtr.Operator != "=" {
			//check if inequality operation has been used in previous filters
			//disallow the filter if inequality was performed on a different field before
			//track the field on which the inequality operation is performed
			if inequalityField != "" && inequalityField != filtr.Field {
				return "", nil, endpoints.BadRequestError
			} else {
				inequalityField = filtr.Field
			}
		}
		
		formattedFilters = append(formattedFilters, filtr)
	}

	return inequalityField, formattedFilters, nil
}