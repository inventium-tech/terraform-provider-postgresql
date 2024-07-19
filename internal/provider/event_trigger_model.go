package provider

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/lib/pq"
	"strings"
	"terraform-provider-postgresql/internal/client"
	"time"
)

type eventTriggerDSModel struct {
	Name     types.String `tfsdk:"name"`
	Event    types.String `tfsdk:"event"`
	Tags     types.Set    `tfsdk:"tags"`
	ExecFunc types.String `tfsdk:"exec_func"`
	Enabled  types.Bool   `tfsdk:"enabled"`
	Database types.String `tfsdk:"database"`
	Owner    types.String `tfsdk:"owner"`
	Comment  types.String `tfsdk:"comment"`
}

type eventTriggerResModel struct {
	Id          types.String `tfsdk:"id"`
	LastUpdated types.String `tfsdk:"last_updated"`
	Name        types.String `tfsdk:"name"`
	Event       types.String `tfsdk:"event"`
	Tags        types.Set    `tfsdk:"tags"`
	ExecFunc    types.String `tfsdk:"exec_func"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Database    types.String `tfsdk:"database"`
	Owner       types.String `tfsdk:"owner"`
	Comment     types.String `tfsdk:"comment"`
}

var (
	eventTriggerEventOptions = []string{
		"ddl_command_start",
		"ddl_command_end",
		"sql_drop",
		"table_rewrite",
	}
)

func (m *eventTriggerResModel) String() string {
	output := `
		Id:          %s
		Name:        %s
		Event:       %s
		Tags:        %v
		ExecFunc:    %s
		Enabled:     %t
		Database:    %s
		Owner:       %s
		Comment:     %s
		LastUpdated: %s
`
	return fmt.Sprintf(
		output,
		m.Id.ValueString(),
		m.Name.ValueString(),
		m.Event.ValueString(),
		m.GetTagsSlice(),
		m.ExecFunc.ValueString(),
		m.Enabled.ValueBool(),
		m.Database.ValueString(),
		m.Owner.ValueString(),
		m.Comment.ValueString(),
		m.LastUpdated.ValueString(),
	)
}

func (m *eventTriggerResModel) SetId() {
	m.Id = types.StringValue(fmt.Sprintf("%s.%s", m.Database.ValueString(), m.Name.ValueString()))
}

func (m *eventTriggerResModel) SetLastUpdated() {
	m.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
}

func (m *eventTriggerResModel) GetTagsSlice() []string {
	tags := make([]string, 0)
	for _, element := range m.Tags.Elements() {
		tags = append(tags, element.(types.String).ValueString())
	}
	return tags
}

func (m *eventTriggerResModel) Create(ctx context.Context, c client.PGClient) diag.Diagnostics {
	diags := diag.Diagnostics{}

	txn, err := c.CreateTransaction(ctx, m.Database.ValueString())
	if err != nil {
		diags.AddError("Error starting transaction", err.Error())
		return diags
	}
	defer c.DeferredRollback(txn)

	createQuery := c.CreateEventTriggerQuery(
		m.Name.ValueString(),
		m.Event.ValueString(),
		m.ExecFunc.ValueString(),
		m.GetTagsSlice(),
	)
	commentQuery := c.CreateCommentQuery("EVENT TRIGGER", m.Name.ValueString(), m.Comment.ValueString())

	if _, err := txn.ExecContext(ctx, createQuery); err != nil {
		diags.AddError("Error creating event trigger", err.Error())
		return diags
	}
	if _, err := txn.ExecContext(ctx, commentQuery); err != nil {
		diags.AddError("Error commenting event trigger", err.Error())
		return diags
	}

	if err = txn.Commit(); err != nil {
		diags.AddError("Error committing transaction", err.Error())
		return diags
	}

	return diags
}

func (m *eventTriggerResModel) Read(ctx context.Context, c client.PGClient) diag.Diagnostics {
	diags := diag.Diagnostics{}

	if m.Id.IsUnknown() || m.Id.IsNull() {
		diags.AddError("Missing Identifier for the event trigger", "Id is required for reading event trigger")
		return diags
	}

	ids := strings.Split(m.Id.ValueString(), ".")
	if len(ids) != 2 || ids[0] == "" || ids[1] == "" {
		diags.AddError("Invalid Identifier for the event trigger", "Id should be in the format 'database_name.event_trigger_name'")
		return diags
	}

	dbName := ids[0]
	eventTriggerName := ids[1]

	conn, err := c.GetConnection(ctx, dbName)
	if err != nil {
		diags.AddError("Error getting connection", err.Error())
		return diags
	}

	query := c.GetEventTriggerQuery(eventTriggerName)

	tflog.Info(ctx, fmt.Sprintf("Query: %s", query))

	var owner, comment, event, execFunc, evtEnabled sql.NullString
	var tags pq.StringArray

	err = conn.QueryRowContext(ctx, query).Scan(&owner, &comment, &event, &tags, &evtEnabled, &execFunc)
	if err != nil {
		diags.AddError(fmt.Sprintf("Error reading event_trigger: '%s'", m.Name.ValueString()), err.Error())
		return diags
	}

	parsedTypes, diags := types.SetValueFrom(ctx, types.StringType, tags)
	if diags.HasError() {
		return diags
	}

	m.Name = types.StringValue(eventTriggerName)
	m.Database = types.StringValue(dbName)
	m.Owner = types.StringValue(owner.String)
	m.Comment = types.StringValue(comment.String)
	m.Event = types.StringValue(event.String)
	m.ExecFunc = types.StringValue(execFunc.String)
	m.Tags = parsedTypes
	// evtenabled: Controls in which session_replication_role modes the event trigger fires.
	// O = trigger fires in “origin” and “local” modes
	// D = trigger is disabled
	// R = trigger fires in “replica” mode
	// A = trigger fires always.
	m.Enabled = types.BoolValue(evtEnabled.String != "D")

	return diags
}

func (m *eventTriggerResModel) Update(ctx context.Context, c client.PGClient, newM *eventTriggerResModel) diag.Diagnostics {
	diags := diag.Diagnostics{}

	txn, err := c.CreateTransaction(ctx, newM.Database.ValueString())
	if err != nil {
		diags.AddError("Error starting transaction", err.Error())
		return diags
	}
	defer c.DeferredRollback(txn)

	if !m.Name.Equal(newM.Name) {
		alterNameQuery := c.AlterObjectNameQuery("EVENT TRIGGER", m.Name.ValueString(), newM.Name.ValueString())
		if _, err := txn.ExecContext(ctx, alterNameQuery); err != nil {
			diags.AddError(fmt.Sprintf("Error renaming event trigger from '%s' to '%s'", m.Name, newM.Name), err.Error())
			return diags
		}
	}

	if !m.Owner.Equal(newM.Owner) {
		alterOwnerQuery := c.AlterObjectOwnerQuery("EVENT TRIGGER", newM.Name.ValueString(), newM.Owner.ValueString())
		if _, err := txn.ExecContext(ctx, alterOwnerQuery); err != nil {
			diags.AddError(fmt.Sprintf("Error changing event trigger's owner from '%s' to '%s'", m.Owner, newM.Owner), err.Error())
			return diags
		}
	}

	if !m.Comment.Equal(newM.Comment) {
		commentQuery := c.CreateCommentQuery("EVENT TRIGGER", newM.Name.ValueString(), newM.Comment.ValueString())
		if _, err := txn.ExecContext(ctx, commentQuery); err != nil {
			diags.AddError("Error commenting event trigger", err.Error())
			return diags
		}
	}

	if !m.Enabled.Equal(newM.Enabled) {
		enabledQuery := c.UpdateEventTriggerEnableQuery(newM.Name.ValueString(), newM.Enabled.ValueBool())
		if _, err := txn.ExecContext(ctx, enabledQuery); err != nil {
			diags.AddError(fmt.Sprintf("Error updating the 'enabled' status for the event trigger '%s'", newM.Name), err.Error())
			return diags
		}
	}

	if err = txn.Commit(); err != nil {
		diags.AddError("Error committing transaction", err.Error())
		return diags
	}

	return diags
}

func (m *eventTriggerResModel) Delete(ctx context.Context, c client.PGClient) diag.Diagnostics {
	diags := diag.Diagnostics{}

	txn, err := c.CreateTransaction(ctx, m.Database.ValueString())
	if err != nil {
		diags.AddError("Error starting transaction", err.Error())
		return diags
	}
	defer c.DeferredRollback(txn)

	dropQuery := c.DropObjectQuery("EVENT TRIGGER", m.Name.ValueString())

	if _, err := txn.ExecContext(ctx, dropQuery); err != nil {
		diags.AddError(fmt.Sprintf("Error dropping event trigger '%s'", m.Name), err.Error())
		return diags
	}

	if err = txn.Commit(); err != nil {
		diags.AddError("Error committing transaction", err.Error())
		return diags
	}

	return diags
}
