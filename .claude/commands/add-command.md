Add a new CLI subcommand to kai named `$ARGUMENTS`.

Steps:

1. Read `internal/cli/root.go` — all commands use unexported constructors (lowercase), registered via `root.AddCommand(new<Name>Cmd())`

2. Read a simple existing command (e.g. `internal/cli/ask.go`) for the pattern: load config inside `RunE` via `loadConfig()`, then call `buildRouter(cfg)` if LLM access is needed

3. Create `internal/cli/$ARGUMENTS.go`:
   ```go
   func new$ARGUMENTSCmd() *cobra.Command {
       cmd := &cobra.Command{
           Use:   "$ARGUMENTS",
           Short: "...",
           RunE: func(cmd *cobra.Command, args []string) error {
               cfg, err := loadConfig()
               if err != nil {
                   return err
               }
               // implementation here
               return nil
           },
       }
       // add flags here with cmd.Flags()
       return cmd
   }
   ```
   Note: constructor is **unexported** (lowercase), takes no parameters.

4. Register in `internal/cli/root.go` inside `NewRootCmd()`:
   ```go
   root.AddCommand(new$ARGUMENTSCmd())
   ```

5. Run `make build && ./kai $ARGUMENTS --help` to verify

Show me `internal/cli/root.go` and one existing command before writing any code.
