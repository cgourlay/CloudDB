/*
 * Copyright (c) 2015 Joern Rischmueller (joern.rm@gmail.com)
 *
 *  This program is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU Affero General Public License as
 *  published by the Free Software Foundation, either version 3 of the
 *  License, or (at your option) any later version.
 *
 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU Affero General Public License for more details.
 *
 *  You should have received a copy of the GNU Affero General Public License
 *  along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */


package goldencheetah

import (
	"net/http"
	"strconv"
	"time"
	"fmt"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"

	"github.com/emicklei/go-restful"
)


// ---------------------------------------------------------------------------------------------------------------//
// Golden Cheetah curator (statusentity) which is stored in DB
// ---------------------------------------------------------------------------------------------------------------//
type StatusEntity struct {
	Status     int    // use Status code 100 = ok, 200 = Partial Failure, 300 = Service Down
	ChangeDate time.Time
}

// ---------------------------------------------------------------------------------------------------------------//
// API View Definition
// ---------------------------------------------------------------------------------------------------------------//

// Full structure for GET and PUT
type StatusEntityAPIv1 struct {
	Id         int64	`json:"id"`
	Status     int     	`json:"status"`
	ChangeDate string      	`json:"changeDate"`
}

type StatusEntityAPIv1List []StatusEntityAPIv1


// ---------------------------------------------------------------------------------------------------------------//
// Data Storage View
// ---------------------------------------------------------------------------------------------------------------//

const statusDBEntity = "statusentity"
const statusDBEntityRootKey = "statusroot"

func mapAPItoDBStatus(api *StatusEntityAPIv1, db *StatusEntity) {
	db.Status = api.Status
	if api.ChangeDate != "" {
		db.ChangeDate, _ = time.Parse(dateTimeLayout, api.ChangeDate)
	} else {
		db.ChangeDate = time.Now()
	}

}

func mapDBtoAPIStatus(db *StatusEntity, api *StatusEntityAPIv1) {
	api.Status = db.Status
	api.ChangeDate = db.ChangeDate.Format(dateTimeLayout)
}


// supporting functions

// curatorEntityKey returns the key used for all curatorEntity entries.
func statusEntityRootKey(ctx context.Context) *datastore.Key {
	return datastore.NewKey(ctx, statusDBEntity, statusDBEntityRootKey, 0, nil)
}

// ---------------------------------------------------------------------------------------------------------------//
// request/response handler
// ---------------------------------------------------------------------------------------------------------------//

func insertStatus(request *restful.Request, response *restful.Response) {
	ctx := appengine.NewContext(request.Request)

	status := new(StatusEntityAPIv1)
	if err := request.ReadEntity(status); err != nil {
		addPlainTextError(response, http.StatusInternalServerError, err.Error())
		return
	}

	// No checks if the necessary fields are filed or not - since GoldenCheetah is
	// the only consumer of the APIs - any checks/response are to support this use-case

	statusDB := new(StatusEntity)
	mapAPItoDBStatus(status, statusDB)

	// and now store it
	key := datastore.NewIncompleteKey(ctx, statusDBEntity, statusEntityRootKey(ctx))
	key, err := datastore.Put(ctx, key, statusDB);
	if err != nil {
		if appengine.IsOverQuota(err) {
			// return 503 and a text similar to what GAE is returning as well
			addPlainTextError(response, http.StatusServiceUnavailable, "503 - Over Quota")
		} else {
			addPlainTextError(response, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// send back the key
	response.WriteHeaderAndEntity(http.StatusCreated, strconv.FormatInt(key.IntID(), 10))

}

func getStatus(request *restful.Request, response *restful.Response) {
	ctx := appengine.NewContext(request.Request)

	var date time.Time
	var err error
	if dateString := request.QueryParameter("dateFrom"); dateString != "" {
		date, err = time.Parse(time.RFC3339, dateString)
		if err != nil {
			addPlainTextError(response, http.StatusBadRequest, fmt.Sprint(err.Error(), " - Correct format is RFC3339"))
			return
		}
	} else {
		date = time.Time{}
	}

	q := datastore.NewQuery(statusDBEntity).Filter("ChangeDate >=", date).Order("-ChangeDate")

	var statusList StatusEntityAPIv1List

	var statusOnDBList []StatusEntity
	k, err := q.GetAll(ctx, &statusOnDBList)
	if err != nil && !isErrFieldMismatch(err) {
		if appengine.IsOverQuota(err) {
			// return 503 and a text similar to what GAE is returning as well
			addPlainTextError(response, http.StatusServiceUnavailable, "503 - Over Quota")
		} else {
			addPlainTextError(response, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// DB Entity needs to be mapped back
	for i, statusDB := range statusOnDBList {
		var statusAPI StatusEntityAPIv1
		mapDBtoAPIStatus(&statusDB, &statusAPI)
		statusAPI.Id = k[i].IntID()
		statusList = append(statusList, statusAPI)
	}

	response.WriteHeaderAndEntity(http.StatusOK, statusList)
}

func getCurrentStatus(request *restful.Request, response *restful.Response) {
	ctx := appengine.NewContext(request.Request)

	q := datastore.NewQuery(statusDBEntity).Order("-ChangeDate").Limit(1)

	var statusList StatusEntityAPIv1List

	var statusOnDBList []StatusEntity
	k, err := q.GetAll(ctx, &statusOnDBList)
	if err != nil && !isErrFieldMismatch(err) {
		if appengine.IsOverQuota(err) {
			// return 503 and a text similar to what GAE is returning as well
			addPlainTextError(response, http.StatusServiceUnavailable, "503 - Over Quota")
		} else {
			addPlainTextError(response, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// DB Entity needs to be mapped back
	var statusAPI StatusEntityAPIv1
	mapDBtoAPIStatus(&statusOnDBList[0], &statusAPI)
	statusAPI.Id = k[0].IntID()
	statusList = append(statusList, statusAPI)

	response.WriteHeaderAndEntity(http.StatusOK, statusList)
}



