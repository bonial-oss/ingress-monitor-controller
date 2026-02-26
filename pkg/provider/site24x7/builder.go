package site24x7

import (
	site24x7 "github.com/Bonial-International-GmbH/site24x7-go"
	site24x7api "github.com/Bonial-International-GmbH/site24x7-go/api"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/config"
	"github.com/bonial-oss/ingress-monitor-controller/pkg/models"
)

type builder struct {
	client     site24x7.Client
	defaults   config.Site24x7MonitorDefaults
	finalizers []finalizer
}

func newBuilder(client site24x7.Client, defaults config.Site24x7MonitorDefaults) *builder {
	b := &builder{
		client:   client,
		defaults: defaults,
	}

	b.finalizers = []finalizer{
		b.finalizeLocationProfile,
		b.finalizeNotificationProfile,
		b.finalizeThresholdProfile,
		b.finalizeMonitorGroup,
		b.finalizeUserGroup,
	}

	return b
}

func (b *builder) FromModel(model *models.Monitor) (*site24x7api.Monitor, error) {
	anno := model.Annotations
	defaults := b.defaults

	monitor := &site24x7api.Monitor{
		Type:        "URL",
		MonitorID:   model.ID,
		DisplayName: model.Name,
		Website:     model.URL,
	}

	monitor.CheckFrequency = anno.StringValue(config.AnnotationSite24x7CheckFrequency, defaults.CheckFrequency)
	monitor.HTTPMethod = anno.StringValue(config.AnnotationSite24x7HTTPMethod, defaults.HTTPMethod)
	monitor.AuthUser = anno.StringValue(config.AnnotationSite24x7AuthUser, defaults.AuthUser)
	monitor.AuthPass = anno.StringValue(config.AnnotationSite24x7AuthPass, defaults.AuthPass)
	monitor.MatchCase = anno.BoolValue(config.AnnotationSite24x7MatchCase, defaults.MatchCase)
	monitor.UserAgent = anno.StringValue(config.AnnotationSite24x7UserAgent, defaults.UserAgent)
	monitor.Timeout = anno.IntValue(config.AnnotationSite24x7Timeout, defaults.Timeout)
	monitor.UseNameServer = anno.BoolValue(config.AnnotationSite24x7UseNameServer, defaults.UseNameServer)
	monitor.UserGroupIDs = anno.StringSliceValue(config.AnnotationSite24x7UserGroupIDs, defaults.UserGroupIDs)
	monitor.MonitorGroups = anno.StringSliceValue(config.AnnotationSite24x7MonitorGroupIDs, defaults.MonitorGroupIDs)
	monitor.LocationProfileID = anno.StringValue(config.AnnotationSite24x7LocationProfileID, defaults.LocationProfileID)
	monitor.NotificationProfileID = anno.StringValue(config.AnnotationSite24x7NotificationProfileID, defaults.NotificationProfileID)
	monitor.ThresholdProfileID = anno.StringValue(config.AnnotationSite24x7ThresholdProfileID, defaults.ThresholdProfileID)

	err := anno.ParseJSON(config.AnnotationSite24x7CustomHeaders, &monitor.CustomHeaders)
	if err != nil {
		return nil, err
	}

	if monitor.CustomHeaders == nil {
		monitor.CustomHeaders = defaults.CustomHeaders
	}

	err = anno.ParseJSON(config.AnnotationSite24x7Actions, &monitor.ActionIDs)
	if err != nil {
		return nil, err
	}

	if monitor.ActionIDs == nil {
		monitor.ActionIDs = defaults.Actions
	}

	return b.finalizeMonitor(monitor)
}

func (b *builder) finalizeMonitor(monitor *site24x7api.Monitor) (*site24x7api.Monitor, error) {
	for _, f := range b.finalizers {
		if err := f(monitor); err != nil {
			return nil, err
		}
	}

	return monitor, nil
}
