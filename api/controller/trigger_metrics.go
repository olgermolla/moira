package controller

import (
	"fmt"
	"net/http"

	"github.com/go-graphite/carbonapi/expr"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/checker"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/target"
)

// GetTriggerMetrics gets all trigger metrics values, default values from: now - 10min, to: now
func GetTriggerMetrics(dataBase moira.Database, from, to int64, triggerID string) (*dto.TriggerMetrics, *api.ErrorResponse) {
	trigger, err := dataBase.GetTrigger(triggerID)
	if err != nil {
		if err == database.ErrNil {
			return nil, api.ErrorInvalidRequest(fmt.Errorf("trigger not found"))
		}
		return nil, api.ErrorInternalServer(err)
	}

	triggerMetrics := dto.TriggerMetrics{
		Main:       make(map[string][]*moira.MetricValue),
		Additional: make(map[string][]*moira.MetricValue),
	}

	isSimpleTrigger := trigger.IsSimple()
	for i, tar := range trigger.Targets {
		result, err := target.EvaluateTarget(dataBase, tar, from, to, isSimpleTrigger)
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}
		for _, timeSeries := range result.TimeSeries {
			values := make([]*moira.MetricValue, 0)
			for i := 0; i < len(timeSeries.Values); i++ {
				timestamp := int64(timeSeries.StartTime + int32(i)*timeSeries.StepTime)
				value := timeSeries.GetTimestampValue(timestamp)
				if !checker.IsInvalidValue(value) {
					values = append(values, &moira.MetricValue{Value: value, Timestamp: timestamp})
				}
			}
			if i == 0 {
				triggerMetrics.Main[timeSeries.Name] = values
			} else {
				triggerMetrics.Additional[timeSeries.Name] = values
			}
		}
	}
	return &triggerMetrics, nil
}

// GetTriggerMetricsPNG gets all trigger metrics values, default values from: now - 10min, to: now
func GetTriggerMetricsPNG(dataBase moira.Database, from, to int64, triggerID string) ([]byte, *api.ErrorResponse) {
	trigger, err := dataBase.GetTrigger(triggerID)
	if err != nil {
		if err == database.ErrNil {
			return nil, api.ErrorInvalidRequest(fmt.Errorf("trigger not found"))
		}
		return nil, api.ErrorInternalServer(err)
	}

	isSimpleTrigger := trigger.IsSimple()
	for _, tar := range trigger.Targets {
		result, err := target.EvaluateTarget(dataBase, tar, from, to, isSimpleTrigger)
		if err != nil {
			return nil, api.ErrorInternalServer(err)
		}

		var metricsData = make([]*expr.MetricData, 0, len(result.TimeSeries))
		for _, ts := range result.TimeSeries {
			metricsData = append(metricsData, &ts.MetricData)
		}
		return expr.MarshalPNG(&http.Request{}, metricsData), nil
	}
	return nil, nil
}

// DeleteTriggerMetric deletes metric from last check and all trigger patterns metrics
func DeleteTriggerMetric(dataBase moira.Database, metricName string, triggerID string) *api.ErrorResponse {
	trigger, err := dataBase.GetTrigger(triggerID)
	if err != nil {
		if err == database.ErrNil {
			return api.ErrorInvalidRequest(fmt.Errorf("trigger not found"))
		}
		return api.ErrorInternalServer(err)
	}

	if err = dataBase.AcquireTriggerCheckLock(triggerID, 10); err != nil {
		return api.ErrorInternalServer(err)
	}
	defer dataBase.DeleteTriggerCheckLock(triggerID)

	lastCheck, err := dataBase.GetTriggerLastCheck(triggerID)
	if err != nil {
		if err == database.ErrNil {
			return api.ErrorInvalidRequest(fmt.Errorf("trigger check not found"))
		}
		return api.ErrorInternalServer(err)
	}
	_, ok := lastCheck.Metrics[metricName]
	if ok {
		delete(lastCheck.Metrics, metricName)
		lastCheck.UpdateScore()
	}
	if err = dataBase.RemovePatternsMetrics(trigger.Patterns); err != nil {
		return api.ErrorInternalServer(err)
	}
	if err = dataBase.SetTriggerLastCheck(triggerID, &lastCheck); err != nil {
		return api.ErrorInternalServer(err)
	}
	return nil
}
