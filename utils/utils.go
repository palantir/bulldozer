// Copyright 2017 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"net/http"

	"github.com/labstack/echo"

	"github.com/palantir/bulldozer/log"
)

func CustomHTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}
	log.FromContext(c).Error(err)
	if err := c.JSON(code, err.Error()); err != nil {
		log.FromContext(c).Error(err)
	}
}

func StringInSlice(element string, list []string) bool {
	for _, e := range list {
		if element == e {
			return true
		}
	}

	return false
}
