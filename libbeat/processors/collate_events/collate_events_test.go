package collate_events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
)

func GetProcessors(t *testing.T, yml []map[string]interface{}) *processors.Processors {
	config := processors.PluginConfig{}

	for _, action := range yml {
		c := map[string]*common.Config{}

		for name, actionYml := range action {
			actionConfig, err := common.NewConfigFrom(actionYml)
			assert.Nil(t, err)

			c[name] = actionConfig
		}
		config = append(config, c)

	}

	list, err := processors.New(config)
	assert.Nil(t, err)

	return list
}

func TestCollateEvents(t *testing.T) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	yml := []map[string]interface{}{
		{
			"collate_events": map[string]interface{}{
				"collation_interval_sec": 5,
				"rules": map[string]interface{}{
					"rule0":map[string]interface{}{
						"when": map[string]interface{}{
							"equals": map[string]string{
								"type": "process",
							},
						},
					},
					"rule1" :map[string]interface{}{
						"when": map[string]interface{}{
							"and" : []map[string]interface{} {
								{
									"contains": map[string]string{
										"proc.name": "test",
									},},
								{
									"regexp" : map[string]string{
										"proc.cmdline": "^launchd",
									},},
							},
						},
					},
					"rule2" :map[string]interface{}{
						"when": map[string]interface{}{
							"or" : []map[string]interface{} {
								{
									"contains": map[string]string{
										"proc.name": "test",
									},},
								{
									"regexp" : map[string]string{
										"proc.cmdline": "^launchd",
									},},
							},
						},
					},
					"rule3" :map[string]interface{}{
						"when": map[string]interface{}{
							"and" : []map[string]interface{} {
								{
									"contains": map[string]string{
										"proc.name": "test",
									},},
								{
									"regexp" : map[string]string{
										"proc.cmdline": "^launchd",
									},},
							},
						},
					},

				},
			},
		},
	}

	processors := GetProcessors(t, yml)

	tm := time.Now()
	event := &beat.Event{
		Timestamp: tm,
		Fields: common.MapStr{
			"beat": common.MapStr{
				"hostname": "mar",
				"name":     "my-shipper-1",
			},
			"proc": common.MapStr{
				"cpu": common.MapStr{
					"start_time": "Jan14",
					"system":     26027,
					"total":      79390,
					"total_p":    0,
					"user":       53363,
				},
				"name":    "test-1",
				"cmdline": "/sbin/launchd",
				"mem": common.MapStr{
					"rss":   11194368,
					"rss_p": 0,
					"share": 0,
					"size":  int64(2555572224),
				},
			},
			"type": "process",
		},
	}

	processedEvent := processors.Run(event)
	assert.Equal(t, event, processedEvent)

	// burst of similar events
	for i := 0; i<1000; i++ {
		event.Timestamp = time.Now()
		processedEvent := processors.Run(event)
		assert.Equal(t, (*beat.Event)(nil), processedEvent)
	}

	// similar events at lower pace
	for i := 0; i<100; i++ {
		tm = tm.Add(time.Second * 2)
		event.Timestamp = tm
		processedEvent := processors.Run(event)
		//assert.Equal(t, (*beat.Event)(nil), processedEvent)
		assert.Equal(t, event, processedEvent)
	}

	event.Timestamp = time.Now()
	processedEvent = processors.Run(event)

	// burst of similar events again
	for i := 0; i<1000; i++ {
		event.Timestamp = time.Now()
		processedEvent := processors.Run(event)
		assert.Equal(t, (*beat.Event)(nil), processedEvent)
	}

}