package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/eegfaktura/eegfaktura-backend/api/middleware"
	"github.com/eegfaktura/eegfaktura-backend/database"
	"github.com/eegfaktura/eegfaktura-backend/graph/generated"
	"github.com/eegfaktura/eegfaktura-backend/graph/gmodel"
	"github.com/eegfaktura/eegfaktura-backend/model"
	log "github.com/sirupsen/logrus"
)

// UpdateEegModel is the resolver for the updateEegModel field.
func (r *mutationResolver) UpdateEegModel(ctx context.Context, tenant string, eegModel *gmodel.EegModel) (*model.Eeg, error) {
	eeg, err := database.GetEeg(database.GetDBXConnection, tenant)
	if err != nil {
		return nil, err
	}

	if eegModel.SettlementInterval != nil {
		eeg.SettlementInterval = *eegModel.SettlementInterval
	}
	if eegModel.SepaActiv != nil {
		eeg.AccountInfo.Sepa = *eegModel.SepaActiv
	}

	return eeg, nil
}

// MasterDataUpload is the resolver for the masterDataUpload field.
func (r *mutationResolver) MasterDataUpload(ctx context.Context, tenant string, sheet string, file graphql.Upload) (bool, error) {

	if err := database.ImportMasterdataFromExcel(database.GetDBXConnection, file.File, file.Filename, sheet, tenant); err != nil {
		return false, err
	}

	return true, nil
}

// Eeg is the resolver for the eeg field.
func (r *queryResolver) Eeg(ctx context.Context) (*model.Eeg, error) {
	tenant := middleware.ForContextTenant(ctx)
	log.Infof("Query Tenant: %+v", tenant)
	eeg, err := database.GetEeg(database.GetDBXConnection, tenant)
	if err != nil {
		return nil, err
	}
	return eeg, nil
}

// Links is the resolver for the links field.
func (r *queryResolver) Links(ctx context.Context) ([]*gmodel.Link, error) {
	panic(fmt.Errorf("not implemented: Links - links"))
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//   - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//     it when you're done.
//   - You have helper methods in this file. Move them out to keep these resolver files clean.
func (r *queryResolver) EegModel(ctx context.Context) (*gmodel.EegModel, error) {
	panic(fmt.Errorf("not implemented: EegModel - eegModel"))
}
