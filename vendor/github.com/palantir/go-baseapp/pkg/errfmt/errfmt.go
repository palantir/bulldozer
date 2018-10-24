// Copyright 2018 Palantir Technologies, Inc.
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

package errfmt

import (
	"fmt"

	"github.com/pkg/errors"
)

type causer interface {
	Cause() error
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func Print(err error) string {
	if err == nil {
		return ""
	}

	var deepestStack stackTracer
	currErr := err
	for currErr != nil {
		if st, ok := currErr.(stackTracer); ok {
			deepestStack = st
		}
		cause, ok := currErr.(causer)
		if !ok {
			break
		}
		currErr = cause.Cause()
	}

	if deepestStack == nil {
		return err.Error()
	}

	return fmt.Sprintf("%s%+v", err.Error(), deepestStack.StackTrace())
}
