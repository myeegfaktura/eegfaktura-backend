package database

import (
	dbsql "database/sql"
	"encoding/json"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/eegfaktura/eegfaktura-backend/model"
)

// NextType is the cursor descriptor emitted in the History.next slot of
// the paginated processhistory response. Mirrors the prod-vfeeg-backend
// shape so customer-web's getHistories1() round-trip matches.
type NextType struct {
	Start    int64  `json:"start,omitempty"`
	End      int64  `json:"end,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	PageSize uint   `json:"page_size,omitempty"`
}

func SaveEdaHistory(dbOpen OpenDbXConnection, history *model.EdaProcessHistory) error {
	db, err := dbOpen()
	if err != nil {
		return err
	}
	defer db.Close()

	sql, _, err := goqu.Insert("base.processhistory").Rows(history).ToSQL()
	_, err = db.Exec(sql)
	return err
}

// FetchEdaHistory returns processhistory rows grouped by protocol, then
// conversation-id. Optional filters: startMs/endMs (epoch ms; 0 means
// "no bound"); protocols (empty means "any non-null protocol", mirroring
// the previous behaviour).
func FetchEdaHistory(dbOpen OpenDbXConnection, tenant string, startMs, endMs int64, protocols []string) (map[string]map[string][]model.EdaProcessHistory, error) {
	db, err := dbOpen()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	where := []goqu.Expression{goqu.C("tenant").Eq(tenant)}
	if len(protocols) > 0 {
		where = append(where, goqu.C("protocol").In(protocols))
	} else {
		where = append(where, goqu.C("protocol").IsNotNull())
	}
	if startMs > 0 {
		where = append(where, goqu.C("date").Gte(time.UnixMilli(startMs)))
	}
	if endMs > 0 {
		where = append(where, goqu.C("date").Lte(time.UnixMilli(endMs)))
	}

	h := []model.EdaProcessHistory{}
	sql, _, err := pgDialect.From("base.processhistory").Select(&h).
		Where(where...).ToSQL()
	if err != nil {
		return nil, err
	}

	err = db.Select(&h, sql)
	if err != nil && err != dbsql.ErrNoRows {
		return nil, err
	}

	out := map[string]map[string][]model.EdaProcessHistory{}
	for _, e := range h {
		_ = json.Unmarshal(e.MessageByte, &e.MessageMap)
		if ci, ok := out[e.Protocol.String]; ok {
			ci[e.ConversationId] = append(ci[e.ConversationId], e)
		} else {
			ci := map[string][]model.EdaProcessHistory{}
			ci[e.ConversationId] = []model.EdaProcessHistory{e}
			out[e.Protocol.String] = ci
		}
	}

	return out, nil
}
