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
	"context"
	"fmt"
	"net/http"
	"sync"

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
	mux         *http.ServeMux
	uploading   bool
	done        bool
	doneChan    chan struct{}
	mutex       sync.Mutex
}

// NewUploadServer returns a new instance of uploadServerApp
func NewUploadServer(bindAddress string, bindPort uint16, pvcDir, destination string) UploadServer {
	server := &uploadServerApp{
		bindAddress: bindAddress,
		bindPort:    bindPort,
		pvcDir:      pvcDir,
		destination: destination,
		mux:         http.NewServeMux(),
		uploading:   false,
		done:        false,
		doneChan:    make(chan struct{}),
	}
	server.mux.HandleFunc(uploadPath, server.uploadHandler)
	return server
}

func (app *uploadServerApp) Run() error {
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", app.bindAddress, app.bindPort),
		Handler: app,
	}

	errChan := make(chan error)

	go func() {
		errChan <- server.ListenAndServe()
	}()

	var err error

	select {
	case err = <-errChan:
		glog.Error("HTTP server returned error %s", err.Error())
	case <-app.doneChan:
		glog.Info("Shutting down http server after successful upload")
		server.Shutdown(context.Background())
	}

	return err
}

func (app *uploadServerApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.mux.ServeHTTP(w, r)
}

func (app *uploadServerApp) uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	app.mutex.Lock()
	exit := func() bool {
		defer app.mutex.Unlock()

		if app.uploading {
			w.WriteHeader(http.StatusServiceUnavailable)
			return true
		}

		if app.done {
			w.WriteHeader(http.StatusConflict)
			return true
		}

		app.uploading = true
		return false
	}()

	if exit {
		glog.Warning("Got concurrent upload request")
		return
	}

	sz, err := importer.SaveStream(r.Body, app.destination)

	app.mutex.Lock()
	defer app.mutex.Unlock()

	if err != nil {
		glog.Errorf("Saving stream failed: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		app.uploading = false
		return
	}

	app.uploading = false
	app.done = true

	close(app.doneChan)

	glog.Infof("Wrote %d bytes to %s", sz, app.destination)
}
