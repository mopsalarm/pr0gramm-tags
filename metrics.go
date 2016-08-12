package main

import "github.com/rcrowley/go-metrics"

var metricsWriteLock = metrics.GetOrRegisterTimer("tags.writelock", nil)
var metricsReadLock = metrics.GetOrRegisterTimer("tags.readlock", nil)
var metricsUpdaterKeysChanged = metrics.GetOrRegisterCounter("tags.updater.keys.changed", nil)
var metricsUpdaterError = metrics.GetOrRegisterCounter("tags.updater.error", nil)
var metricsKeysCount = metrics.GetOrRegisterGauge("tags.keys.count", nil)
var metricsSearch = metrics.GetOrRegisterTimer("tags.search", nil)
var metricsCheckpointError = metrics.GetOrRegisterCounter("tags.checkpoint.error", nil)

