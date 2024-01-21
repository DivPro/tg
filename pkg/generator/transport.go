// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (transport.go at 24.06.2020, 15:26) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package generator

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"

	"github.com/sirupsen/logrus"

	"github.com/seniorGolang/tg/v2/pkg/astra"
	"github.com/seniorGolang/tg/v2/pkg/astra/types"
	"github.com/seniorGolang/tg/v2/pkg/mod"
	"github.com/seniorGolang/tg/v2/pkg/tags"
)

const keyCode = "code"

const doNotEdit = "GENERATED BY 'T'ransport 'G'enerator. DO NOT EDIT."

const (
	tagLogger              = "log"
	tagDesc                = "desc"
	tagType                = "type"
	tagTag                 = "tags"
	tagTests               = "tests"
	tagTrace               = "trace"
	tagEnums               = "enums"
	tagFormat              = "format"
	tagSummary             = "summary"
	tagHandler             = "handler"
	tagExample             = "example"
	tagMetrics             = "metrics"
	tagHttpArg             = "http-args"
	tagHttpPath            = "http-path"
	tagDeprecated          = "deprecated"
	tagHttpPrefix          = "http-prefix"
	tagMethodHTTP          = "http-method"
	tagServerHTTP          = "http-server"
	tagHttpHeader          = "http-headers"
	tagHttpCookies         = "http-cookies"
	tagHttpSuccess         = "http-success"
	tagServerJsonRPC       = "jsonRPC-server"
	tagHttpResponse        = "http-response"
	tagPackageJSON         = "packageJSON"
	tagPackageUUID         = "uuidPackage"
	tagSwaggerTags         = "swaggerTags"
	tagLogSkip             = "log-skip"
	tagDisableOmitEmpty    = "tagNoOmitempty"
	tagRequestContentType  = "requestContentType"
	tagResponseContentType = "responseContentType"

	tagTitle       = "title"
	tagNameNPM     = "npmName"
	tagServers     = "servers"
	tagSecurity    = "security"
	tagAppVersion  = "version"
	tagAuthor      = "author"
	tagLicense     = "license"
	tagPrivateNPM  = "npmPrivate"
	tagRegistryNPM = "npmRegistry"
)

type Transport struct {
	hasJsonRPC bool
	version    string
	modPath    string
	tags       tags.DocTags
	module     *modfile.File
	log        logrus.FieldLogger
	services   map[string]*service
}

func NewTransport(log logrus.FieldLogger, version, svcDir string, ifaces ...string) (tr Transport, err error) {

	tr.log = log
	tr.version = version
	var files []os.DirEntry
	tr.services = make(map[string]*service)
	var include, exclude []string
	for _, iface := range ifaces {
		if strings.HasPrefix(iface, "!") {
			exclude = append(exclude, strings.TrimPrefix(iface, "!"))
			continue
		}
		include = append(include, iface)
	}
	if len(include) != 0 && len(exclude) != 0 {
		err = fmt.Errorf("include and exclude cannot be set at same time")
		return
	}
	if err = tr.goMod(svcDir); err != nil {
		return
	}
	if files, err = os.ReadDir(svcDir); err != nil {
		return
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") {
			continue
		}
		var serviceAst *types.File
		svcDir, _ = filepath.Abs(svcDir)
		filePath := path.Join(svcDir, file.Name())
		if serviceAst, err = astra.ParseFile(filePath); err != nil {
			return
		}
		tr.tags = tr.tags.Merge(tags.ParseTags(serviceAst.Docs))
		for _, iface := range serviceAst.Interfaces {
			if len(include) != 0 {
				if !slices.Contains(include, iface.Name) {
					log.WithField("iface", iface.Name).Info("skip")
					continue
				}
			}
			if len(exclude) != 0 {
				if slices.Contains(exclude, iface.Name) {
					log.WithField("iface", iface.Name).Info("skip")
					continue
				}
			}
			if len(tags.ParseTags(iface.Docs)) != 0 {
				service := newService(log, &tr, filePath, iface)
				tr.services[iface.Name] = service
				if service.tags.Contains(tagServerJsonRPC) {
					tr.hasJsonRPC = true
				}
			}
		}
	}
	return
}

func (tr *Transport) RenderAzure(appName, routePrefix, outDir, logLevel string, enableHealth bool) (err error) {
	return newAzure(tr).render(appName, routePrefix, outDir, logLevel, enableHealth)
}

func (tr *Transport) RenderSwagger(outDir string, interfaces ...string) (err error) {
	return newSwagger(tr).render(outDir, interfaces...)
}

func (tr *Transport) serviceKeys() (keys []string) {

	for serviceName := range tr.services {
		keys = append(keys, serviceName)
	}
	sort.Strings(keys)
	return
}

func (tr *Transport) RenderClient(outDir string) (err error) {

	tr.cleanup(outDir)
	if err = os.MkdirAll(outDir, 0777); err != nil {
		return
	}

	if tr.hasTrace() {
		showError(tr.log, tr.renderClientTracer(outDir), "renderClientTracer")
	}
	showError(tr.log, tr.renderClientOptions(outDir), "renderClientOptions")
	if tr.hasJsonRPC {
		showError(tr.log, tr.renderVersion(outDir, false), "renderVersion")
		showError(tr.log, tr.renderClientJsonRPC(outDir), "renderClientJsonRPC")
		showError(tr.log, tr.renderClientError(outDir), "renderClientError")
		showError(tr.log, tr.renderClientBatch(outDir), "renderClientBatch")
		showError(tr.log, tr.renderClientCache(outDir), "renderClientCache")
	}
	for _, serviceName := range tr.serviceKeys() {
		svc := tr.services[serviceName]
		showError(tr.log, svc.renderClient(outDir), "renderHTTP")
	}
	return
}

func (tr *Transport) RenderServer(outDir string) (err error) {

	tr.cleanup(outDir)

	if err = os.MkdirAll(outDir, 0777); err != nil {
		return
	}

	hasTrace := tr.hasTrace()
	hasMetric := tr.hasMetrics()

	showError(tr.log, tr.renderHTTP(outDir), "renderHTTP")
	showError(tr.log, tr.renderFiber(outDir), "renderFiber")
	showError(tr.log, tr.renderHeader(outDir), "renderHeader")
	showError(tr.log, tr.renderErrors(outDir), "renderErrors")
	showError(tr.log, tr.renderServer(outDir), "renderServer")
	showError(tr.log, tr.renderOptions(outDir), "renderOptions")
	showError(tr.log, tr.renderVersion(outDir, false), "renderVersion")
	if hasMetric {
		showError(tr.log, tr.renderMetrics(outDir), "renderMetrics")
	}
	if hasTrace {
		showError(tr.log, tr.renderTracer(outDir), "renderTracer")
	}
	if tr.hasJsonRPC {
		showError(tr.log, tr.renderJsonRPC(outDir), "renderJsonRPC")
	}

	for _, serviceName := range tr.serviceKeys() {
		svc := tr.services[serviceName]
		err = svc.render(outDir)
	}
	return
}

func (tr *Transport) hasTrace() (hasTrace bool) {
	for _, serviceName := range tr.serviceKeys() {
		svc := tr.services[serviceName]
		if svc.tags.IsSet(tagTrace) {
			return true
		}
	}
	return
}

func (tr *Transport) hasMetrics() (hasMetric bool) {
	for _, serviceName := range tr.serviceKeys() {
		svc := tr.services[serviceName]
		if svc.tags.IsSet(tagMetrics) {
			return true
		}
	}
	return
}

func showError(log logrus.FieldLogger, err error, msg string) {
	if err != nil {
		log.WithError(err).Error(msg)
	}
}

func (tr *Transport) goMod(svcDir string) (err error) {

	if tr.modPath, err = mod.GoModPath(svcDir); err != nil {
		return err
	}
	var fileBytes []byte
	if fileBytes, err = os.ReadFile(tr.modPath); err != nil {
		return
	}
	tr.module, err = modfile.Parse("go.mod", fileBytes, nil)
	return
}
