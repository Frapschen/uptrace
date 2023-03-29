package metrics

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/uptrace/pkg/bunapp"
	"github.com/uptrace/uptrace/pkg/metrics/upql"
)

const (
	DashGrid  = "grid"
	DashTable = "table"
)

type DashGauge struct {
	bun.BaseModel `bun:"dash_gauges,alias:g"`

	ID uint64 `json:"id,string" bun:",pk,autoincrement"`

	ProjectID uint32     `json:"projectId"`
	DashID    uint64     `json:"dashId,string"`
	Dash      *Dashboard `json:"-" bun:"rel:belongs-to,on_delete:CASCADE"`
	DashKind  string     `json:"-"`

	Name        string `json:"name"`
	Description string `json:"description"`
	Weight      int    `json:"weight"`
	Template    string `json:"template"`

	Metrics []upql.Metric            `json:"metrics"`
	Query   string                   `json:"query"`
	Columns map[string]*MetricColumn `json:"columnMap" bun:",nullzero"`
}

func (g *DashGauge) Validate() error {
	if g.Name == "" {
		return fmt.Errorf("gauge name can't be empty")
	}
	if err := g.validate(); err != nil {
		return fmt.Errorf("gauge %q is invalid: %w", g.Name, err)
	}
	return nil
}

func (g *DashGauge) validate() error {
	if g.ProjectID == 0 {
		return fmt.Errorf("project id can't be zero")
	}
	if g.DashID == 0 {
		return fmt.Errorf("dash id can't be zero")
	}
	if g.DashKind == "" {
		return fmt.Errorf("dashb kind can't be empty")
	}
	if g.Description == "" {
		return fmt.Errorf("description can't be empty")
	}
	if len(g.Metrics) == 0 {
		return fmt.Errorf("at least one metric is required")
	}

	if g.Query == "" {
		return fmt.Errorf("query can't be empty")
	}
	if err := upql.Validate(g.Query); err != nil {
		return fmt.Errorf("can't parse query: %w", err)
	}

	return nil
}

func SelectTableGridGauges(
	ctx context.Context, app *bunapp.App, dashID uint64,
) (table, grid []*DashGauge, _ error) {
	gauges, err := SelectDashGauges(ctx, app, dashID)
	if err != nil {
		return nil, nil, err
	}

	table = make([]*DashGauge, 0)
	grid = make([]*DashGauge, 0)

	for _, gauge := range gauges {
		switch gauge.DashKind {
		case DashTable:
			table = append(table, gauge)
		case DashGrid:
			grid = append(grid, gauge)
		default:
			return nil, nil, fmt.Errorf("unknown dashboard kind: %q", gauge.DashKind)
		}
	}

	return table, grid, nil
}

func SelectDashGauges(
	ctx context.Context, app *bunapp.App, dashID uint64,
) ([]*DashGauge, error) {
	var gauges []*DashGauge
	if err := app.PG.NewSelect().
		Model(&gauges).
		Where("dash_id = ?", dashID).
		OrderExpr("weight DESC, id ASC").
		Scan(ctx); err != nil {
		return nil, err
	}
	return gauges, nil
}

func InsertDashGauges(ctx context.Context, app *bunapp.App, gauges []*DashGauge) error {
	if len(gauges) == 0 {
		return nil
	}

	for _, entry := range gauges {
		if entry.Columns == nil {
			entry.Columns = make(map[string]*MetricColumn)
		}
	}

	if _, err := app.PG.NewInsert().
		Model(&gauges).
		Exec(ctx); err != nil {
		return err
	}
	return nil
}
