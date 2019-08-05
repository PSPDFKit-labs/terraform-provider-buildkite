package buildkite

import (
	"context"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/shurcooL/graphql"
)

type RESTPipeline struct {
	Slug                string                 `json:"slug,omitempty"`
	BranchConfiguration string                 `json:"branch_configuration"`        // Can be empty.
	Provider            SourceProvider         `json:"provider,omitempty"`          // Used when reading a pipeline.
	ProviderSettings    SourceProviderSettings `json:"provider_settings,omitempty"` // Used when creating and updating a pipeline.
}

// Adapted from https://github.com/buildkite/go-buildkite/blob/568b6651b687ccf6893ada08086ce58b072538b6/buildkite/providers.go.
type SourceProvider struct {
	Settings SourceProviderSettings `json:"settings"`
}

// Adapted from https://github.com/buildkite/go-buildkite/blob/568b6651b687ccf6893ada08086ce58b072538b6/buildkite/providers.go.
type SourceProviderSettings struct {
	TriggerMode                             string `json:"trigger_mode,omitempty"`
	BuildPullRequests                       bool   `json:"build_pull_requests,omitempty"`
	PullRequestBranchFilterEnabled          bool   `json:"pull_request_branch_filter_enabled,omitempty"`
	PullRequestBranchFilterConfiguration    string `json:"pull_request_branch_filter_configuration"` // Can be empty.
	SkipPullRequestBuildsForExistingCommits bool   `json:"skip_pull_request_builds_for_existing_commits,omitempty"`
	BuildPullRequestForks                   bool   `json:"build_pull_request_forks,omitempty"`
	PrefixPullRequestForkBranchNames        bool   `json:"prefix_pull_request_fork_branch_names,omitempty"`
	BuildTags                               bool   `json:"build_tags,omitempty"`
	PublishCommitStatus                     bool   `json:"publish_commit_status,omitempty"`
	PublishCommitStatusPerStep              bool   `json:"publish_commit_status_per_step,omitempty"`
	SeparatePullRequestStatuses             bool   `json:"separate_pull_request_statuses,omitempty"`
	PublishBlockedAsPending                 bool   `json:"publish_blocked_as_pending,omitempty"`
}

// PipelineNode represents a pipeline as returned from the GraphQL API
type PipelineNode struct {
	DefaultBranch graphql.String
	Description   graphql.String
	Id            graphql.String
	Name          graphql.String
	Repository    struct {
		Url graphql.String
	}
	Slug  graphql.String
	Steps struct {
		Yaml graphql.String
	}
	Uuid                                 graphql.String
	WebhookURL                           graphql.String `graphql:"webhookURL"`
	SkipIntermediateBuilds               graphql.Boolean
	SkipIntermediateBuildsBranchFilter   graphql.String
	CancelIntermediateBuilds             graphql.Boolean
	CancelIntermediateBuildsBranchFilter graphql.String
}

func resourcePipeline() *schema.Resource {
	return &schema.Resource{
		Create: CreatePipeline,
		Read:   ReadPipeline,
		Update: UpdatePipeline,
		Delete: DeletePipeline,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Required: true,
				Type:     schema.TypeString,
			},
			"description": &schema.Schema{
				Optional: true,
				Type:     schema.TypeString,
			},
			"repository": &schema.Schema{
				Required: true,
				Type:     schema.TypeString,
			},
			"webhook_url": &schema.Schema{
				Computed: true,
				Type:     schema.TypeString,
			},
			"slug": &schema.Schema{
				Computed: true,
				Type:     schema.TypeString,
			},
			"steps": &schema.Schema{
				Required: true,
				Type:     schema.TypeString,
			},
			"default_branch": &schema.Schema{
				Optional: true,
				Type:     schema.TypeString,
			},
			"skip_intermediate_builds": &schema.Schema{
				Optional: true,
				Type:     schema.TypeBool,
			},
			"skip_intermediate_builds_branch_filter": &schema.Schema{
				Optional: true,
				Type:     schema.TypeString,
			},
			"cancel_intermediate_builds": &schema.Schema{
				Optional: true,
				Type:     schema.TypeBool,
			},
			"cancel_intermediate_builds_branch_filter": &schema.Schema{
				Optional: true,
				Type:     schema.TypeString,
			},
			"branch_configuration": &schema.Schema{
				Optional: true,
				Type:     schema.TypeString,
			},
			// provider_settings:
			// (adapted from https://github.com/yougroupteam/terraform-buildkite/blob/0846b1b73fef1eb2930b047c7156cdf1b4dde3fe/buildkite/resource_pipeline.go#L190-L251)
			"trigger_mode": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "code",
			},
			"build_pull_requests": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"pull_request_branch_filter_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"pull_request_branch_filter_configuration": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"skip_pull_request_builds_for_existing_commits": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"build_pull_request_forks": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"prefix_pull_request_fork_branch_names": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"build_tags": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"publish_commit_status": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"publish_commit_status_per_step": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"separate_pull_request_statuses": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"publish_blocked_as_pending": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

// CreatePipeline creates a Buildkite pipeline
func CreatePipeline(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)
	id, err := GetOrganizationID(client.organization, client.graphql)
	if err != nil {
		return err
	}

	var mutation struct {
		PipelineCreate struct {
			Pipeline PipelineNode
		} `graphql:"pipelineCreate(input: {organizationId: $org, name: $name, description: $desc, repository: {url: $repository_url}, steps: {yaml: $steps}, defaultBranch: $default_branch, skipIntermediateBuilds: $skip_intermediate_builds, skipIntermediateBuildsBranchFilter: $skip_intermediate_builds_branch_filter, cancelIntermediateBuilds: $cancel_intermediate_builds, cancelIntermediateBuildsBranchFilter: $cancel_intermediate_builds_branch_filter})"`
	}

	vars := map[string]interface{}{
		"desc":                                   graphql.String(d.Get("description").(string)),
		"name":                                   graphql.String(d.Get("name").(string)),
		"org":                                    id,
		"repository_url":                         graphql.String(d.Get("repository").(string)),
		"steps":                                  graphql.String(d.Get("steps").(string)),
		"default_branch":                         graphql.String(d.Get("default_branch").(string)),
		"skip_intermediate_builds":               graphql.Boolean(d.Get("skip_intermediate_builds").(bool)),
		"skip_intermediate_builds_branch_filter": graphql.String(d.Get("skip_intermediate_builds_branch_filter").(string)),
		"cancel_intermediate_builds":             graphql.Boolean(d.Get("cancel_intermediate_builds").(bool)),
		"cancel_intermediate_builds_branch_filter": graphql.String(d.Get("cancel_intermediate_builds_branch_filter").(string)),
	}

	err = client.graphql.Mutate(context.Background(), &mutation, vars)
	if err != nil {
		return err
	}

	updatePipeline(d, &mutation.PipelineCreate.Pipeline)

	restPipeline := getRESTPipeline(d)

	err = client.rest.Pipelines.Update(restPipeline)
	if err != nil {
		return err
	}

	updateRESTPipeline(d, restPipeline)

	return nil
}

// ReadPipeline retrieves a Buildkite pipeline
func ReadPipeline(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)
	var query struct {
		Node struct {
			Pipeline PipelineNode `graphql:"... on Pipeline"`
		} `graphql:"node(id: $id)"`
	}

	vars := map[string]interface{}{
		"id": graphql.ID(d.Id()),
	}

	err := client.graphql.Query(context.Background(), &query, vars)
	if err != nil {
		return err
	}

	updatePipeline(d, &query.Node.Pipeline)

	slug := d.Get("slug").(string)
	restPipeline, err := client.rest.Pipelines.Read(slug)
	if err != nil {
		return err
	}

	updateRESTPipeline(d, restPipeline)

	return nil
}

// UpdatePipeline updates a Buildkite pipeline
func UpdatePipeline(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)

	var mutation struct {
		PipelineUpdate struct {
			Pipeline PipelineNode
		} `graphql:"pipelineUpdate(input: {id: $id, name: $name, description: $desc, repository: {url: $repository_url}, steps: {yaml: $steps}, defaultBranch: $default_branch, skipIntermediateBuilds: $skip_intermediate_builds, skipIntermediateBuildsBranchFilter: $skip_intermediate_builds_branch_filter, cancelIntermediateBuilds: $cancel_intermediate_builds, cancelIntermediateBuildsBranchFilter: $cancel_intermediate_builds_branch_filter})"`
	}

	vars := map[string]interface{}{
		"desc":                                   graphql.String(d.Get("description").(string)),
		"id":                                     graphql.ID(d.Id()),
		"name":                                   graphql.String(d.Get("name").(string)),
		"repository_url":                         graphql.String(d.Get("repository").(string)),
		"steps":                                  graphql.String(d.Get("steps").(string)),
		"default_branch":                         graphql.String(d.Get("default_branch").(string)),
		"skip_intermediate_builds":               graphql.Boolean(d.Get("skip_intermediate_builds").(bool)),
		"skip_intermediate_builds_branch_filter": graphql.String(d.Get("skip_intermediate_builds_branch_filter").(string)),
		"cancel_intermediate_builds":             graphql.Boolean(d.Get("cancel_intermediate_builds").(bool)),
		"cancel_intermediate_builds_branch_filter": graphql.String(d.Get("cancel_intermediate_builds_branch_filter").(string)),
	}

	err := client.graphql.Mutate(context.Background(), &mutation, vars)
	if err != nil {
		return err
	}

	updatePipeline(d, &mutation.PipelineUpdate.Pipeline)

	restPipeline := getRESTPipeline(d)

	err = client.rest.Pipelines.Update(restPipeline)
	if err != nil {
		return err
	}

	updateRESTPipeline(d, restPipeline)

	return nil
}

// DeletePipeline removes a Buildkite pipeline
func DeletePipeline(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)
	slug := d.Get("slug").(string)

	// there is no delete mutation in graphql yet so we must use rest api
	err := client.rest.Pipelines.Delete(slug)
	return err
}

func updatePipeline(d *schema.ResourceData, t *PipelineNode) {
	d.SetId(string(t.Id))
	d.Set("description", string(t.Description))
	d.Set("name", string(t.Name))
	d.Set("repository", string(t.Repository.Url))
	d.Set("slug", string(t.Slug))
	d.Set("steps", string(t.Steps.Yaml))
	d.Set("uuid", string(t.Uuid))
	d.Set("webhook_url", string(t.WebhookURL))
	d.Set("default_branch", string(t.DefaultBranch))
	d.Set("skip_intermediate_builds", bool(t.SkipIntermediateBuilds))
	d.Set("skip_intermediate_builds_branch_filter", string(t.SkipIntermediateBuildsBranchFilter))
	d.Set("cancel_intermediate_builds", bool(t.CancelIntermediateBuilds))
	d.Set("cancel_intermediate_builds_branch_filter", string(t.CancelIntermediateBuildsBranchFilter))
}

func updateRESTPipeline(d *schema.ResourceData, p *RESTPipeline) {
	d.Set("branch_configuration", p.BranchConfiguration)
	d.Set("trigger_mode", p.Provider.Settings.TriggerMode)
	d.Set("build_pull_requests", p.Provider.Settings.BuildPullRequests)
	d.Set("pull_request_branch_filter_enabled", p.Provider.Settings.PullRequestBranchFilterEnabled)
	d.Set("pull_request_branch_filter_configuration", p.Provider.Settings.PullRequestBranchFilterConfiguration)
	d.Set("skip_pull_request_builds_for_existing_commits", p.Provider.Settings.SkipPullRequestBuildsForExistingCommits)
	d.Set("build_pull_request_forks", p.Provider.Settings.BuildPullRequestForks)
	d.Set("prefix_pull_request_fork_branch_names", p.Provider.Settings.PrefixPullRequestForkBranchNames)
	d.Set("build_tags", p.Provider.Settings.BuildTags)
	d.Set("publish_commit_status", p.Provider.Settings.PublishCommitStatus)
	d.Set("publish_commit_status_per_step", p.Provider.Settings.PublishCommitStatusPerStep)
	d.Set("separate_pull_request_statuses", p.Provider.Settings.SeparatePullRequestStatuses)
	d.Set("publish_blocked_as_pending", p.Provider.Settings.PublishBlockedAsPending)
}

func getRESTPipeline(d *schema.ResourceData) *RESTPipeline {
	return &RESTPipeline{
		Slug:                d.Get("slug").(string),
		BranchConfiguration: d.Get("branch_configuration").(string),
		ProviderSettings: SourceProviderSettings{
			TriggerMode:                             d.Get("trigger_mode").(string),
			BuildPullRequests:                       d.Get("build_pull_requests").(bool),
			PullRequestBranchFilterEnabled:          d.Get("pull_request_branch_filter_enabled").(bool),
			PullRequestBranchFilterConfiguration:    d.Get("pull_request_branch_filter_configuration").(string),
			SkipPullRequestBuildsForExistingCommits: d.Get("skip_pull_request_builds_for_existing_commits").(bool),
			BuildPullRequestForks:                   d.Get("build_pull_request_forks").(bool),
			PrefixPullRequestForkBranchNames:        d.Get("prefix_pull_request_fork_branch_names").(bool),
			BuildTags:                               d.Get("build_tags").(bool),
			PublishCommitStatus:                     d.Get("publish_commit_status").(bool),
			PublishCommitStatusPerStep:              d.Get("publish_commit_status_per_step").(bool),
			SeparatePullRequestStatuses:             d.Get("separate_pull_request_statuses").(bool),
			PublishBlockedAsPending:                 d.Get("publish_blocked_as_pending").(bool),
		},
	}
}
