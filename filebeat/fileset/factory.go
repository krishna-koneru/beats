package fileset

import (
	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/filebeat/channel"
	pros "github.com/elastic/beats/filebeat/prospector"
	"github.com/elastic/beats/filebeat/registrar"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
)

// Factory for modules
type Factory struct {
	outlet                channel.Factory
	registrar             *registrar.Registrar
	beatVersion           string
	pipelineLoaderFactory PipelineLoaderFactory
	beatDone              chan struct{}
}

// Wrap an array of prospectors and implements cfgfile.Runner interface
type prospectorsRunner struct {
	id                    uint64
	moduleRegistry        *ModuleRegistry
	prospectors           []*pros.Prospector
	pipelineLoaderFactory PipelineLoaderFactory
}

// NewFactory instantiates a new Factory
func NewFactory(outlet channel.Factory, registrar *registrar.Registrar, beatVersion string,
	pipelineLoaderFactory PipelineLoaderFactory, beatDone chan struct{}) *Factory {
	return &Factory{
		outlet:                outlet,
		registrar:             registrar,
		beatVersion:           beatVersion,
		beatDone:              beatDone,
		pipelineLoaderFactory: pipelineLoaderFactory,
	}
}

// Create creates a module based on a config
func (f *Factory) Create(c *common.Config) (cfgfile.Runner, error) {
	// Start a registry of one module:
	m, err := NewModuleRegistry([]*common.Config{c}, f.beatVersion)
	if err != nil {
		return nil, err
	}

	pConfigs, err := m.GetProspectorConfigs()
	if err != nil {
		return nil, err
	}

	// Hash module ID
	var h map[string]interface{}
	c.Unpack(&h)
	id, err := hashstructure.Hash(h, nil)
	if err != nil {
		return nil, err
	}

	prospectors := make([]*pros.Prospector, len(pConfigs))
	for i, pConfig := range pConfigs {
		prospectors[i], err = pros.NewProspector(pConfig, f.outlet, f.beatDone, f.registrar.GetStates())
		if err != nil {
			logp.Err("Error creating prospector: %s", err)
			return nil, err
		}
	}

	return &prospectorsRunner{
		id:                    id,
		moduleRegistry:        m,
		prospectors:           prospectors,
		pipelineLoaderFactory: f.pipelineLoaderFactory,
	}, nil
}

func (p *prospectorsRunner) Start() {
	// Load pipelines
	if p.pipelineLoaderFactory != nil {
		// Load pipelines instantly and then setup a callback for reconnections:
		pipelineLoader, err := p.pipelineLoaderFactory()
		if err != nil {
			logp.Err("Error loading pipeline: %s", err)
		} else {
			err := p.moduleRegistry.LoadPipelines(pipelineLoader)
			if err != nil {
				// Log error and continue
				logp.Err("Error loading pipeline: %s", err)
			}
		}

		// Callback:
		callback := func(esClient *elasticsearch.Client) error {
			return p.moduleRegistry.LoadPipelines(esClient)
		}
		elasticsearch.RegisterConnectCallback(callback)
	}

	for _, prospector := range p.prospectors {
		prospector.Start()
	}
}
func (p *prospectorsRunner) Stop() {
	for _, prospector := range p.prospectors {
		prospector.Stop()
	}
}
func (p *prospectorsRunner) ID() uint64 {
	return p.id
}
