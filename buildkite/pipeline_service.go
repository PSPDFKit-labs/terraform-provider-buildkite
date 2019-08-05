package buildkite

import "fmt"

// Adapted from https://github.com/buildkite/go-buildkite/blob/568b6651b687ccf6893ada08086ce58b072538b6/buildkite/pipelines.go.

type PipelinesService struct {
	client *RESTClient
}

// Create is managed by GraphQL client.

func (ps *PipelinesService) Read(slug string) (*RESTPipeline, error) {
	u := fmt.Sprintf("v2/organizations/%s/pipelines/%s", ps.client.organization, slug)

	req, err := ps.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	pipeline := new(RESTPipeline)
	_, err = ps.client.Do(req, pipeline)
	if err != nil {
		return nil, err
	}

	return pipeline, nil
}

func (ps *PipelinesService) Update(pipeline *RESTPipeline) error {
	u := fmt.Sprintf("v2/organizations/%s/pipelines/%s", ps.client.organization, pipeline.Slug)

	req, err := ps.client.NewRequest("PATCH", u, pipeline)
	if err != nil {
		return err
	}

	_, err = ps.client.Do(req, pipeline)
	return err
}

func (ps *PipelinesService) Delete(slug string) error {
	u := fmt.Sprintf("v2/organizations/%s/pipelines/%s", ps.client.organization, slug)

	req, err := ps.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	_, err = ps.client.Do(req, nil)
	return err
}
