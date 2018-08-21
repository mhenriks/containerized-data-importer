/*
 * This file is part of the CDI project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2018 Red Hat, Inc.
 *
 */

package uploadserver

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"kubevirt.io/containerized-data-importer/pkg/importer"
)

const (
	uploadPath = "/v1alpha1/upload"
)

// UploadServer is the interface to uploadServerApp
type UploadServer interface {
	Run() error
}

type uploadServerApp struct {
	bindAddress string
	bindPort    uint16
	pvcDir      string
	destination string
}

// NewUploadServer returns a new instance of uploadServerApp
func NewUploadServer(bindAddress string, bindPort uint16, pvcDir, destination string) UploadServer {
	return &uploadServerApp{
		bindAddress: bindAddress,
		bindPort:    bindPort,
		pvcDir:      pvcDir,
		destination: destination,
	}
}

func (app *uploadServerApp) Run() error {
	http.HandleFunc(uploadPath, app.uploadHandler)

	return http.ListenAndServe(fmt.Sprintf("%s:%d", app.bindAddress, app.bindPort), nil)
}

func (app *uploadServerApp) uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	sz, err := importer.SaveStream(r.Body, app.destination)
	if err != nil {
		glog.Errorf("Saving stream failed: %s", err)
		return
	}

	glog.Infof("Wrote %d bytes to %s", sz, app.destination)
}
