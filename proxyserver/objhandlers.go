//  Copyright (c) 2015 Rackspace
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
//  implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package proxyserver

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/troubling/hummingbird/common"
	"github.com/troubling/hummingbird/common/srv"
	"github.com/troubling/hummingbird/proxyserver/middleware"
)

func (server *ProxyServer) ObjectGetHandler(writer http.ResponseWriter, request *http.Request) {
	vars := srv.GetVars(request)
	ctx := middleware.GetProxyContext(request)
	if ctx == nil {
		srv.StandardResponse(writer, 500)
		return
	}
	containerInfo := ctx.C.GetContainerInfo(vars["account"], vars["container"])
	if containerInfo == nil {
		srv.StandardResponse(writer, 404)
		return
	}
	ctx.ACL = containerInfo.ReadACL
	if ctx.Authorize != nil && !ctx.Authorize(request) {
		if ctx.RemoteUser != "" {
			srv.StandardResponse(writer, 403)
			return
		}
		srv.StandardResponse(writer, 401)
		return
	}
	r, headers, code := ctx.C.GetObject(vars["account"], vars["container"], vars["obj"], request.Header)
	for k := range headers {
		writer.Header().Set(k, headers.Get(k))
	}
	writer.WriteHeader(code)
	if r != nil {
		defer r.Close()
		common.Copy(r, writer)
	}
}

func (server *ProxyServer) ObjectHeadHandler(writer http.ResponseWriter, request *http.Request) {
	vars := srv.GetVars(request)
	ctx := middleware.GetProxyContext(request)
	if ctx == nil {
		srv.StandardResponse(writer, 500)
		return
	}
	containerInfo := ctx.C.GetContainerInfo(vars["account"], vars["container"])
	if containerInfo == nil {
		srv.StandardResponse(writer, 404)
		return
	}
	ctx.ACL = containerInfo.ReadACL
	if ctx.Authorize != nil && !ctx.Authorize(request) {
		if ctx.RemoteUser != "" {
			srv.StandardResponse(writer, 403)
			return
		}
		srv.StandardResponse(writer, 401)
		return
	}
	headers, code := ctx.C.HeadObject(vars["account"], vars["container"], vars["obj"], request.Header)
	for k := range headers {
		writer.Header().Set(k, headers.Get(k))
	}
	writer.WriteHeader(code)
}

func (server *ProxyServer) ObjectDeleteHandler(writer http.ResponseWriter, request *http.Request) {
	vars := srv.GetVars(request)
	ctx := middleware.GetProxyContext(request)
	if ctx == nil {
		srv.StandardResponse(writer, 500)
		return
	}
	containerInfo := ctx.C.GetContainerInfo(vars["account"], vars["container"])
	if containerInfo == nil {
		srv.StandardResponse(writer, 404)
		return
	}
	ctx.ACL = containerInfo.WriteACL
	if ctx.Authorize != nil && !ctx.Authorize(request) {
		if ctx.RemoteUser != "" {
			srv.StandardResponse(writer, 403)
			return
		}
		srv.StandardResponse(writer, 401)
		return
	}
	srv.StandardResponse(writer, ctx.C.DeleteObject(vars["account"], vars["container"], vars["obj"], request.Header))
}

func (server *ProxyServer) ObjectPutHandler(writer http.ResponseWriter, request *http.Request) {
	vars := srv.GetVars(request)
	ctx := middleware.GetProxyContext(request)
	if ctx == nil {
		srv.StandardResponse(writer, 500)
		return
	}
	if !(request.Header.Get("If-None-Match") == "" ||
		request.Header.Get("If-None-Match") == "*") {
		srv.SimpleErrorResponse(writer, 400, "If-None-Match only supports *")
		return
	}
	containerInfo := ctx.C.GetContainerInfo(vars["account"], vars["container"])
	if containerInfo == nil {
		srv.StandardResponse(writer, 404)
		return
	}
	ctx.ACL = containerInfo.WriteACL
	if ctx.Authorize != nil && !ctx.Authorize(request) {
		if ctx.RemoteUser != "" {
			srv.StandardResponse(writer, 403)
			return
		}
		srv.StandardResponse(writer, 401)
		return
	}
	if request.Header.Get("Content-Type") == "" {
		contentType := mime.TypeByExtension(filepath.Ext(vars["obj"]))
		contentType = strings.Split(contentType, ";")[0] // remove any charset it tried to foist on us
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		request.Header.Set("Content-Type", contentType)
	}
	if status, str := CheckObjPut(request, vars["obj"]); status != http.StatusOK {
		writer.Header().Set("Content-Type", "text/plain")
		writer.WriteHeader(status)
		writer.Write([]byte(str))
		return
	}
	h, code := ctx.C.PutObject(vars["account"], vars["container"], vars["obj"], request.Header, request.Body)
	writer.Header().Set("Etag", h.Get("Etag"))
	if modified, err := common.ParseDate(request.Header.Get("X-Timestamp")); err == nil {
		writer.Header().Set("Last-Modified", common.FormatLastModified(modified))
	}
	srv.StandardResponse(writer, code)
}
