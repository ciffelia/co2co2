package main

import (
	"context"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
)

var configuration = datadog.NewConfiguration()
var apiClient = datadog.NewAPIClient(configuration)
var metricsApi = datadogV2.NewMetricsApi(apiClient)

func SubmitRecord(ctx context.Context, record *Record) error {
	timestamp := datadog.PtrInt64(record.Timestamp.Unix())
	_, _, err := metricsApi.SubmitMetrics(ctx, datadogV2.MetricPayload{
		Series: []datadogV2.MetricSeries{
			{
				Metric: "sensor.ud_co2s.co2",
				Type:   datadogV2.METRICINTAKETYPE_GAUGE.Ptr(),
				// NOTE: omitting unit because ppm is not supported in DataDog: https://docs.datadoghq.com/metrics/units/
				Points: []datadogV2.MetricPoint{
					{
						Timestamp: timestamp,
						Value:     datadog.PtrFloat64(float64(record.Co2)),
					},
				},
			},
			{
				Metric: "sensor.ud_co2s.temperature",
				Type:   datadogV2.METRICINTAKETYPE_GAUGE.Ptr(),
				Unit:   datadog.PtrString("degree celsius"),
				Points: []datadogV2.MetricPoint{
					{
						Timestamp: timestamp,
						Value:     &record.Temperature,
					},
				},
			},
			{
				Metric: "sensor.ud_co2s.humidity",
				Type:   datadogV2.METRICINTAKETYPE_GAUGE.Ptr(),
				Unit:   datadog.PtrString("percent"),
				Points: []datadogV2.MetricPoint{
					{
						Timestamp: timestamp,
						Value:     &record.Humidity,
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
}
