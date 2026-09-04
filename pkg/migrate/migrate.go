// Package migrate applies the application schema on Microsoft SQL Server.
package migrate

import (
	"fmt"
	"slices"
	"strings"

	"dfms/apps/auth"
	"dfms/apps/models"
	"dfms/internal/setup"
	"dfms/internal/workflow"
	"dfms/pkg/logs"
	"dfms/pkg/schema"
	"dfms/pkg/types"

	"gorm.io/gorm"
)

// Bootstrap creates/updates all tables from the GORM models.
func Bootstrap(db *gorm.DB) error {
	for _, g := range schema.Groups() {
		if err := db.AutoMigrate(g.Models...); err != nil {
			return fmt.Errorf("migrate %s: %w", g.Label, err)
		}
		logs.Infof("migrated group %q (%d models)", g.Label, len(g.Models))
	}
	for _, idx := range filteredUniqueIndexes() {
		if err := ensureFilteredUniqueIndex(db, idx); err != nil {
			return err
		}
	}
	if err := dropRetiredDecimalPrecision(db); err != nil {
		return err
	}
	if err := dropRetiredCurrencySageColumns(db); err != nil {
		return err
	}
	if err := allowNullTruckTankTruckID(db); err != nil {
		return err
	}
	return ensureIntegrationSettingKeyCheck(db)
}

// allowNullTruckTankTruckID lets a retired tank keep its plate after the truck
// is unlinked. GORM AutoMigrate does not drop NOT NULL on an existing column.
func allowNullTruckTankTruckID(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.TruckTank{}) || !db.Migrator().HasColumn(&models.TruckTank{}, "TruckID") {
		return nil
	}
	var nullable bool
	if err := db.Raw(`
		SELECT CAST(c.is_nullable AS bit)
		FROM sys.columns c
		WHERE c.object_id = OBJECT_ID(N'[TruckTank]') AND c.name = N'TruckID'
	`).Scan(&nullable).Error; err != nil {
		return fmt.Errorf("TruckTank.TruckID nullability: %w", err)
	}
	if nullable {
		return nil
	}
	var typeName string
	if err := db.Raw(`
		SELECT t.name
		FROM sys.columns c
		INNER JOIN sys.types t ON c.user_type_id = t.user_type_id
		WHERE c.object_id = OBJECT_ID(N'[TruckTank]') AND c.name = N'TruckID'
	`).Scan(&typeName).Error; err != nil {
		return fmt.Errorf("TruckTank.TruckID type: %w", err)
	}
	if typeName == "" {
		typeName = "bigint"
	}
	type fkRow struct {
		Name string
		Ref  string
	}
	var fks []fkRow
	if err := db.Raw(`
		SELECT fk.name, OBJECT_NAME(fk.referenced_object_id) AS ref
		FROM sys.foreign_keys fk
		INNER JOIN sys.foreign_key_columns fkc ON fk.object_id = fkc.constraint_object_id
		INNER JOIN sys.columns c ON fkc.parent_object_id = c.object_id AND fkc.parent_column_id = c.column_id
		WHERE fk.parent_object_id = OBJECT_ID(N'[TruckTank]') AND c.name = N'TruckID'
	`).Scan(&fks).Error; err != nil {
		return fmt.Errorf("list TruckTank.TruckID foreign keys: %w", err)
	}
	for _, fk := range fks {
		if err := db.Exec(fmt.Sprintf("ALTER TABLE [TruckTank] DROP CONSTRAINT %s", quoteMSSQLIdent(fk.Name))).Error; err != nil {
			return fmt.Errorf("drop TruckTank FK %s: %w", fk.Name, err)
		}
	}
	if err := db.Exec(fmt.Sprintf("ALTER TABLE [TruckTank] ALTER COLUMN [TruckID] %s NULL", typeName)).Error; err != nil {
		return fmt.Errorf("allow null TruckTank.TruckID: %w", err)
	}
	for _, fk := range fks {
		sql := fmt.Sprintf(
			"ALTER TABLE [TruckTank] ADD CONSTRAINT %s FOREIGN KEY ([TruckID]) REFERENCES %s ([ID])",
			quoteMSSQLIdent(fk.Name), quoteMSSQLIdent(fk.Ref),
		)
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("restore TruckTank FK %s: %w", fk.Name, err)
		}
	}
	logs.Info("TruckTank.TruckID is now nullable")
	return nil
}

// dropRetiredCurrencySageColumns removes Sage 200 numeric ids. Currencies are
// identified by ISO code (same key Sage 300 uses).
func dropRetiredDecimalPrecision(db *gorm.DB) error {
	if !db.Migrator().HasTable("DecimalPrecision") {
		return nil
	}
	if err := db.Exec("DROP TABLE IF EXISTS [DecimalPrecision]").Error; err != nil {
		return fmt.Errorf("drop DecimalPrecision: %w", err)
	}
	logs.Infof("dropped retired table DecimalPrecision")
	return nil
}

func dropRetiredCurrencySageColumns(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.Currency{}) {
		return nil
	}
	var checks []string
	if err := db.Raw(`
		SELECT name FROM sys.check_constraints
		WHERE parent_object_id = OBJECT_ID(N'[Currency]')
		  AND definition LIKE '%SageID%'
	`).Scan(&checks).Error; err != nil {
		return fmt.Errorf("list Currency SageID checks: %w", err)
	}
	for _, name := range checks {
		if err := db.Exec(fmt.Sprintf("ALTER TABLE [Currency] DROP CONSTRAINT %s", quoteMSSQLIdent(name))).Error; err != nil {
			logs.Warnf("drop Currency check %s: %v", name, err)
		}
	}
	var indexes []string
	if err := db.Raw(`
		SELECT DISTINCT i.name
		FROM sys.indexes i
		INNER JOIN sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id
		INNER JOIN sys.columns c ON ic.object_id = c.object_id AND ic.column_id = c.column_id
		WHERE i.object_id = OBJECT_ID(N'[Currency]')
		  AND c.name IN (N'SageID', N'SageAliases')
		  AND i.is_primary_key = 0
		  AND i.name IS NOT NULL
	`).Scan(&indexes).Error; err != nil {
		return fmt.Errorf("list Currency Sage indexes: %w", err)
	}
	for _, name := range indexes {
		if err := db.Exec(fmt.Sprintf("DROP INDEX %s ON [Currency]", quoteMSSQLIdent(name))).Error; err != nil {
			logs.Warnf("drop Currency index %s: %v", name, err)
		}
	}
	for _, col := range []string{"SageID", "SageAliases"} {
		if !db.Migrator().HasColumn(&models.Currency{}, col) {
			continue
		}
		if err := db.Migrator().DropColumn(&models.Currency{}, col); err != nil {
			return fmt.Errorf("drop Currency.%s: %w", col, err)
		}
		logs.Infof("dropped Currency.%s", col)
	}
	return nil
}

func ensureIntegrationSettingKeyCheck(db *gorm.DB) error {
	var names []string
	if err := db.Raw(`
		SELECT name FROM sys.check_constraints
		WHERE parent_object_id = OBJECT_ID(N'[IntegrationSetting]')
		  AND definition LIKE '%[Key]%'
	`).Scan(&names).Error; err != nil {
		return fmt.Errorf("list IntegrationSetting checks: %w", err)
	}
	for _, name := range names {
		if err := db.Exec(fmt.Sprintf("ALTER TABLE [IntegrationSetting] DROP CONSTRAINT %s", quoteMSSQLIdent(name))).Error; err != nil {
			logs.Warnf("drop IntegrationSetting check %s: %v", name, err)
		}
	}
	quoted := make([]string, 0, len(types.IntegrationKeys))
	for _, k := range types.IntegrationKeys {
		quoted = append(quoted, "'"+k+"'")
	}
	inList := strings.Join(quoted, ",")
	if err := db.Exec(fmt.Sprintf("DELETE FROM [IntegrationSetting] WHERE [Key] NOT IN (%s)", inList)).Error; err != nil {
		return fmt.Errorf("remove unknown integration settings: %w", err)
	}
	const chk = "chk_IntegrationSetting_Key"
	_ = db.Exec(fmt.Sprintf("ALTER TABLE [IntegrationSetting] DROP CONSTRAINT %s", quoteMSSQLIdent(chk)))
	sql := fmt.Sprintf(`ALTER TABLE [IntegrationSetting] ADD CONSTRAINT [chk_IntegrationSetting_Key]
		CHECK ([Key] IN (%s))`, inList)
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("add IntegrationSetting key check: %w", err)
	}
	return nil
}

type filteredUniqueIndex struct {
	Name     string
	Table    string
	Columns  []string
	WhereSQL string
}

func filteredUniqueIndexes() []filteredUniqueIndex {
	return []filteredUniqueIndex{
		{
			Name:     "idx_uniqueProfilePhone",
			Table:    "Profile",
			Columns:  []string{"PhoneNumber"},
			WhereSQL: "[PhoneNumber] <> ''",
		},
		{
			Name:     "idx_uniqueActiveInstanceDoc",
			Table:    "ProcessInstance",
			Columns:  []string{"DocContentType", "ObjectID"},
			WhereSQL: "[Status] IN ('running', 'draft')",
		},
		{
			Name:     "idx_uniquePendingTaskUser",
			Table:    "Task",
			Columns:  []string{"InstanceID", "NodeID", "UserID"},
			WhereSQL: "[Status] = 'pending' AND [UserID] IS NOT NULL",
		},
		{
			Name:     "idx_uniqueActiveCompIlo",
			Table:    "GantryCompartmentalization",
			Columns:  []string{"IloID"},
			WhereSQL: "[IsActive] = 1 AND [Amended] = 0 AND [Status] <> 'cancelled' AND [Status] <> 'rejected'",
		},
		{
			Name:     "idx_uniqueGLOCustomerOrder",
			Table:    "GantryLoadingLine",
			Columns:  []string{"RequestID", "CustomerOrderNumber"},
			WhereSQL: "[CustomerOrderNumber] <> '' AND [IsActive] = 1",
		},
		{
			Name:     "idx_uniqueCompTopSeal",
			Table:    "GantryCompartmentalizationLine",
			Columns:  []string{"TopSeal"},
			WhereSQL: "[TopSeal] <> ''",
		},
		{
			Name:     "idx_uniqueCompDipSeal",
			Table:    "GantryCompartmentalizationLine",
			Columns:  []string{"DipSeal"},
			WhereSQL: "[DipSeal] <> ''",
		},
		{
			Name:     "idx_uniqueCompBottomSeal",
			Table:    "GantryCompartmentalizationLine",
			Columns:  []string{"BottomSeal"},
			WhereSQL: "[BottomSeal] <> ''",
		},
	}
}

func ensureFilteredUniqueIndex(db *gorm.DB, idx filteredUniqueIndex) error {
	var exists int
	if err := db.Raw(`
	SELECT COUNT(*) FROM sys.indexes
	WHERE name = ? AND object_id = OBJECT_ID(?)`, idx.Name, "["+idx.Table+"]").Scan(&exists).Error; err != nil {
		return fmt.Errorf("check index %s: %w", idx.Name, err)
	}
	if exists > 0 {
		return nil
	}
	cols := make([]string, len(idx.Columns))
	for i, c := range idx.Columns {
		cols[i] = "[" + c + "]"
	}
	sql := fmt.Sprintf(
		`CREATE UNIQUE NONCLUSTERED INDEX [%s] ON [%s](%s) WHERE %s`,
		idx.Name, idx.Table, strings.Join(cols, ", "), idx.WhereSQL,
	)
	if err := db.Exec(sql).Error; err != nil {
		return fmt.Errorf("create %s: %w", idx.Name, err)
	}
	logs.Infof("created filtered unique index %s on %s(%s)", idx.Name, idx.Table, strings.Join(idx.Columns, ", "))
	return nil
}

// Up applies the schema then seeds auth, reference data, and workflows.
func Up(db *gorm.DB) error {
	if err := Bootstrap(db); err != nil {
		return err
	}
	if err := auth.LoadDefault(db); err != nil {
		return fmt.Errorf("seed auth: %w", err)
	}
	if err := setup.SeedReference(db); err != nil {
		return fmt.Errorf("seed reference data: %w", err)
	}
	if err := retireUnusedProcesses(db); err != nil {
		return fmt.Errorf("retire unused workflows: %w", err)
	}
	if err := workflow.Seed(db); err != nil {
		return fmt.Errorf("seed workflow: %w", err)
	}
	logs.Info("seeded auth, reference data, EWURA licenses, and workflows")
	return nil
}

// retireUnusedProcesses removes approval graphs that are not in the TIPER catalogue.
func retireUnusedProcesses(db *gorm.DB) error {
	keep := workflow.SeedProcessCodes()
	if len(keep) == 0 {
		return nil
	}
	var ids []uint
	if err := db.Model(&models.Process{}).Where("Code NOT IN ?", keep).Pluck("ID", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var instIDs []uint
		if err := tx.Model(&models.ProcessInstance{}).Where("ProcessID IN ?", ids).Pluck("ID", &instIDs).Error; err != nil {
			return err
		}
		if len(instIDs) > 0 {
			if err := tx.Where("InstanceID IN ?", instIDs).Delete(&models.Event{}).Error; err != nil {
				return err
			}
			if err := tx.Where("InstanceID IN ?", instIDs).Delete(&models.Task{}).Error; err != nil {
				return err
			}
			if err := tx.Where("ID IN ?", instIDs).Delete(&models.ProcessInstance{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("ProcessID IN ?", ids).Delete(&models.ApprovalSubstitute{}).Error; err != nil {
			return err
		}
		if err := tx.Where("ProcessID IN ?", ids).Delete(&models.WorkflowNotifyUser{}).Error; err != nil {
			return err
		}
		var poolIDs []uint
		if err := tx.Model(&models.InitiatorPool{}).Where("ProcessID IN ?", ids).Pluck("ID", &poolIDs).Error; err != nil {
			return err
		}
		if len(poolIDs) > 0 {
			if err := tx.Where("InitiatorPoolID IN ?", poolIDs).Delete(&models.WorkflowInitiatorPoolUser{}).Error; err != nil {
				return err
			}
			if err := tx.Where("ID IN ?", poolIDs).Delete(&models.InitiatorPool{}).Error; err != nil {
				return err
			}
		}
		var nodeIDs []uint
		if err := tx.Model(&models.Node{}).Where("ProcessID IN ?", ids).Pluck("ID", &nodeIDs).Error; err != nil {
			return err
		}
		if err := tx.Where("ProcessID IN ?", ids).Delete(&models.Transition{}).Error; err != nil {
			return err
		}
		if len(nodeIDs) > 0 {
			if err := tx.Where("NodeID IN ?", nodeIDs).Delete(&models.NodeOperatorRole{}).Error; err != nil {
				return err
			}
			if err := tx.Where("NodeID IN ?", nodeIDs).Delete(&models.NodeOperatorUser{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&models.Process{}).Where("ID IN ?", ids).Update("RejectReturnNodeID", nil).Error; err != nil {
			return err
		}
		if err := tx.Where("ProcessID IN ?", ids).Delete(&models.Node{}).Error; err != nil {
			return err
		}
		if err := tx.Where("ID IN ?", ids).Delete(&models.Process{}).Error; err != nil {
			return err
		}
		logs.Infof("retired unused workflow processes %d", len(ids))
		return nil
	})
}

func IsEmpty(db *gorm.DB) bool {
	return !db.Migrator().HasTable(&models.User{})
}

func RequireReady(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.User{}) {
		return fmt.Errorf("database has no application tables — run schema migration first:\n  dfms migrate up")
	}
	return nil
}

func Reset(db *gorm.DB) error {
	tables, err := managedTableNames(db)
	if err != nil {
		return err
	}
	if err := dropForeignKeys(db, tables); err != nil {
		return err
	}
	for _, name := range slices.Backward(tables) {
		if !db.Migrator().HasTable(name) {
			continue
		}
		sql := fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteMSSQLIdent(name))
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("drop %s: %w", name, err)
		}
		logs.Infof("dropped table %s", name)
	}
	return Up(db)
}

func managedTableNames(db *gorm.DB) ([]string, error) {
	seen := make(map[string]struct{})
	var out []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	for _, m := range schema.AllModels() {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(m); err != nil {
			return nil, fmt.Errorf("parse %T: %w", m, err)
		}
		add(stmt.Schema.Table)
		for _, rel := range stmt.Schema.Relationships.Relations {
			if rel.JoinTable != nil {
				add(rel.JoinTable.Name)
			}
		}
	}
	for _, m := range []any{
		&models.UserRole{}, &models.RolesPermission{},
		&models.NodeOperatorRole{}, &models.NodeOperatorUser{},
		&models.WorkflowInitiatorPoolUser{},
		&models.WorkflowNotifyUser{},
	} {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(m); err != nil {
			return nil, fmt.Errorf("parse join %T: %w", m, err)
		}
		add(stmt.Schema.Table)
	}
	return out, nil
}

type fkRow struct {
	FKName    string `gorm:"column:FKName"`
	TableName string `gorm:"column:TableName"`
	RefTable  string `gorm:"column:RefTable"`
}

func dropForeignKeys(db *gorm.DB, tables []string) error {
	if len(tables) == 0 {
		return nil
	}
	want := make(map[string]struct{}, len(tables))
	for _, t := range tables {
		want[t] = struct{}{}
	}
	var rows []fkRow
	q := `
		SELECT fk.name AS FKName,
			OBJECT_NAME(fk.parent_object_id) AS TableName,
			OBJECT_NAME(fk.referenced_object_id) AS RefTable
		FROM sys.foreign_keys AS fk`
	if err := db.Raw(q).Scan(&rows).Error; err != nil {
		return fmt.Errorf("list foreign keys: %w", err)
	}
	dropped := make(map[string]struct{})
	for _, r := range rows {
		if r.FKName == "" || r.TableName == "" {
			continue
		}
		_, parentManaged := want[r.TableName]
		_, refManaged := want[r.RefTable]
		if !parentManaged && !refManaged {
			continue
		}
		key := r.TableName + "\x00" + r.FKName
		if _, ok := dropped[key]; ok {
			continue
		}
		sql := fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s",
			quoteMSSQLIdent(r.TableName), quoteMSSQLIdent(r.FKName))
		if err := db.Exec(sql).Error; err != nil {
			msg := strings.ToLower(err.Error())
			if strings.Contains(msg, "does not exist") ||
				strings.Contains(msg, "cannot find") ||
				strings.Contains(msg, "is not a constraint") ||
				strings.Contains(msg, "could not drop") {
				dropped[key] = struct{}{}
				continue
			}
			return fmt.Errorf("drop FK %s on %s: %w", r.FKName, r.TableName, err)
		}
		dropped[key] = struct{}{}
	}
	return nil
}

func quoteMSSQLIdent(name string) string {
	return "[" + strings.ReplaceAll(name, "]", "]]") + "]"
}
