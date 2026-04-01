package provider

import (
	"context"
	"fmt"
	"terraform-provider-postgresql/internal/helpers"
	"terraform-provider-postgresql/internal/pgclient"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type resourceModelEventTrigger struct {
	Id          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Event       types.String `tfsdk:"event"`
	Tags        types.Set    `tfsdk:"tags"`
	ExecFunc    types.String `tfsdk:"exec_func"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Database    types.String `tfsdk:"database"`
	Owner       types.String `tfsdk:"owner"`
	Comment     types.String `tfsdk:"comment"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

type datasourceModelEventTrigger struct {
	Name     types.String `tfsdk:"name"`
	Event    types.String `tfsdk:"event"`
	Tags     types.Set    `tfsdk:"tags"`
	ExecFunc types.String `tfsdk:"exec_func"`
	Enabled  types.Bool   `tfsdk:"enabled"`
	Database types.String `tfsdk:"database"`
	Owner    types.String `tfsdk:"owner"`
	Comment  types.String `tfsdk:"comment"`
}

func (m *resourceModelEventTrigger) SetId() {
	m.Id = types.StringValue(fmt.Sprintf("%s.%s", m.Database.ValueString(), m.Name.ValueString()))
}

func readEventTriggerFromDB[M resourceModelEventTrigger | datasourceModelEventTrigger](ctx context.Context, pgConn *pgx.Conn, repo pgclient.EventTriggerRepo, eventName string, model *M) diag.Diagnostics {
	diags := diag.Diagnostics{}

	pgEventTrigger, err := repo.GetOne(ctx, pgConn, eventName)
	if err != nil {
		diags.AddError(fmt.Sprintf(msgErrorExecutingPgAction, TFReadAction, PGEventTrigger), err.Error())
		return diags
	}

	// common attributes
	name := types.StringValue(pgEventTrigger.Name.String)
	event := types.StringValue(pgEventTrigger.Event.String)
	execFunc := types.StringValue(pgEventTrigger.ExecFunc.String)
	enabled := types.BoolValue(pgEventTrigger.Enabled.Bool)
	database := types.StringValue(pgEventTrigger.Database.String)
	owner := types.StringValue(pgEventTrigger.Owner.String)
	comment := types.StringValue(pgEventTrigger.Comment.String)

	tagsSlice := helpers.SliceMap(pgEventTrigger.Tags.Elements, func(tag pgtype.Text) attr.Value {
		return types.StringValue(tag.String)
	})

	tags, diags := types.SetValue(types.StringType, tagsSlice)
	if diags.HasError() {
		return diags
	}

	switch m := any(model).(type) {
	case *resourceModelEventTrigger:
		m.Name = name
		m.Event = event
		m.ExecFunc = execFunc
		m.Tags = tags
		m.Enabled = enabled
		m.Database = database
		m.Owner = owner
		m.Comment = comment
	case *datasourceModelEventTrigger:
		m.Name = name
		m.Event = event
		m.ExecFunc = execFunc
		m.Tags = tags
		m.Enabled = enabled
		m.Database = database
		m.Owner = owner
		m.Comment = comment
	}

	return diags
}
