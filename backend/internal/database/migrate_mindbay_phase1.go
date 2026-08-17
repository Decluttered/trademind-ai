package database

import (
	"fmt"
	"gorm.io/gorm"
)

func migrateMindBayPhase1Immutability(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	switch db.Dialector.Name() {
	case "postgres":
		if err := db.Exec(`CREATE OR REPLACE FUNCTION mindbay_reject_immutable_write() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'MindBay immutable row cannot be changed'; END; $$ LANGUAGE plpgsql`).Error; err != nil {
			return err
		}
		for _, table := range []string{"product_snapshot", "listing_content_version"} {
			for _, op := range []string{"UPDATE", "DELETE"} {
				name := fmt.Sprintf("trg_%s_no_%s", table, op)
				if err := db.Exec(fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON %s", name, table)).Error; err != nil {
					return err
				}
				if err := db.Exec(fmt.Sprintf("CREATE TRIGGER %s BEFORE %s ON %s FOR EACH ROW EXECUTE FUNCTION mindbay_reject_immutable_write()", name, op, table)).Error; err != nil {
					return err
				}
			}
		}
	case "sqlite":
		for _, table := range []string{"product_snapshot", "listing_content_version"} {
			for _, op := range []string{"UPDATE", "DELETE"} {
				name := fmt.Sprintf("trg_%s_no_%s", table, op)
				sql := fmt.Sprintf("CREATE TRIGGER IF NOT EXISTS %s BEFORE %s ON %s BEGIN SELECT RAISE(ABORT, 'MindBay immutable row cannot be changed'); END", name, op, table)
				if err := db.Exec(sql).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}
