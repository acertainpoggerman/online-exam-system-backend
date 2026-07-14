package users

import (
	"context"
	"errors"
	"log"

	"github.com/acertainpoggerman/online-exam-system/internal/apperr"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

func (svc *authService) resolveRegisterError(ctx context.Context, err error) error {

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		log.Printf("Error in RegisterUser: could not create examiner (%v)\n", err)
		return apperr.ErrInternal
	}

	// Email already in Use
	if pgErr.Code == pgerrcode.UniqueViolation {

		// Get the column involved
		col, colErr := svc.getColumnForConstraint(ctx, pgErr.ConstraintName)
		if colErr != nil {
			log.Printf("Error in RegisterUser: failed to get column (%v)\n", colErr)
			return apperr.ErrInternal
		}

		// If it isn't email, log and return server error
		if col == "email" {
			return &apperr.FieldError{
				Fields:  []string{col},
				Message: "Email already in use",
				Code:    "EMAIL_TAKEN",
			}
		}
		log.Printf("Error in RegisterUser: Unique constraint violated for col (%s)\n", col)
		return apperr.ErrInternal

	}

	// Check in DB failed
	if pgErr.Code == pgerrcode.CheckViolation {

		// Get the column involved
		col, colErr := svc.getColumnForConstraint(ctx, pgErr.ConstraintName)
		if colErr != nil {
			log.Printf("Error in RegisterUser: failed to get column (%v)\n", colErr)
			return apperr.ErrInternal
		}

		if col != "email" {
			return &apperr.FieldError{
				Fields:  []string{col},
				Message: "Please fill out required fields",
				Code:    "REQUIRED_FIELDS",
			}
		}

		return &apperr.FieldError{
			Fields:  []string{col},
			Message: "Invalid email passed.",
			Code:    "INVALID_EMAIL",
		}

	}

	log.Printf("Error in RegisterUser: could not create examiner (pgError: %v)\n", pgErr)
	return apperr.ErrInternal // If PostgreSQL error isn't covered
}

func (svc *authService) getColumnForConstraint(ctx context.Context, constraintName string) (string, error) {
	var columnName string
	query := `
        SELECT column_name
        FROM information_schema.constraint_column_usage
        WHERE constraint_name = $1
    `
	err := svc.pool.QueryRow(ctx, query, constraintName).Scan(&columnName)
	return columnName, err
}
