package cli

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/pyroscope-io/pyroscope/pkg/config"
	scrape "github.com/pyroscope-io/pyroscope/pkg/scrape/config"
	"github.com/pyroscope-io/pyroscope/pkg/scrape/discovery"
	sm "github.com/pyroscope-io/pyroscope/pkg/scrape/model"
	"github.com/pyroscope-io/pyroscope/pkg/util/bytesize"
)

type FlagsStruct struct {
	Config   string            `mapstructure:"config"`
	Foo      string            `mapstructure:"foo"`
	Foos     []string          `mapstructure:"foos"`
	Bar      int               `mapstructure:"bar"`
	Baz      time.Duration     `mapstructure:"baz"`
	FooBar   string            `mapstructure:"foo-bar"`
	FooFoo   float64           `mapstructure:"foo-foo"`
	FooBytes bytesize.ByteSize `mapstructure:"foo-bytes"`
	FooDur   time.Duration     `mapstructure:"foo-dur"`
}

var _ = Describe("flags", func() {
	Context("PopulateFlagSet", func() {
		Context("without config file", func() {
			It("correctly sets all types of arguments", func() {
				vpr := viper.New()
				exampleCommand := &cobra.Command{
					RunE: func(cmd *cobra.Command, args []string) error {
						return nil
					},
				}

				cfg := FlagsStruct{}
				PopulateFlagSet(&cfg, exampleCommand.Flags(), vpr)

				b := bytes.NewBufferString("")
				exampleCommand.SetOut(b)
				exampleCommand.SetArgs([]string{
					fmt.Sprintf("--foo=%s", "test-val-1"),
					fmt.Sprintf("--foos=%s", "test-val-2"),
					fmt.Sprintf("--foos=%s", "test-val-3"),
					fmt.Sprintf("--bar=%s", "123"),
					fmt.Sprintf("--baz=%s", "10h"),
					fmt.Sprintf("--foo-bar=%s", "test-val-4"),
					fmt.Sprintf("--foo-foo=%s", "10.23"),
					fmt.Sprintf("--foo-bytes=%s", "100MB"),
				})

				err := exampleCommand.Execute()
				Expect(err).ToNot(HaveOccurred())
				Expect(cfg.Foo).To(Equal("test-val-1"))
				Expect(cfg.Foos).To(Equal([]string{"test-val-2", "test-val-3"}))
				Expect(cfg.Bar).To(Equal(123))
				Expect(cfg.Baz).To(Equal(10 * time.Hour))
				Expect(cfg.FooBar).To(Equal("test-val-4"))
				Expect(cfg.FooFoo).To(Equal(10.23))
				Expect(cfg.FooBytes).To(Equal(100 * bytesize.MB))
			})
		})

		Context("with config file", func() {
			It("correctly sets all types of arguments", func() {
				cfg := FlagsStruct{}
				vpr := viper.New()
				exampleCommand := &cobra.Command{
					RunE: CreateCmdRunFn(&cfg, vpr, func(cmd *cobra.Command, args []string) error {
						return nil
					}),
				}

				PopulateFlagSet(&cfg, exampleCommand.Flags(), vpr)
				vpr.BindPFlags(exampleCommand.Flags())

				b := bytes.NewBufferString("")
				exampleCommand.SetOut(b)
				exampleCommand.SetArgs([]string{fmt.Sprintf("--config=%s", "testdata/example.yml")})

				err := exampleCommand.Execute()
				Expect(err).ToNot(HaveOccurred())
				Expect(cfg.Foo).To(Equal("test-val-1"))
				Expect(cfg.Foos).To(Equal([]string{"test-val-2", "test-val-3"}))
				Expect(cfg.Bar).To(Equal(123))
				Expect(cfg.Baz).To(Equal(10 * time.Hour))
				Expect(cfg.FooBar).To(Equal("test-val-4"))
				Expect(cfg.FooFoo).To(Equal(10.23))
				Expect(cfg.FooBytes).To(Equal(100 * bytesize.MB))
				Expect(cfg.FooDur).To(Equal(5*time.Minute + 23*time.Second))
			})

			It("correctly works with substitutions", func() {
				os.Setenv("VALUE1", "test-val-1")
				os.Setenv("VALUE2", "test-val-2")
				// os.Setenv("VALUE3", "test-val-3")
				os.Setenv("VALUE4", "123")
				os.Setenv("VALUE5", "10h")
				os.Setenv("VALUE6", "test-val-4")
				os.Setenv("VALUE7", "10.23")
				os.Setenv("VALUE8", "100mb")
				os.Setenv("VALUE9", "5m23s")
				cfg := FlagsStruct{}
				vpr := viper.New()

				exampleCommand := &cobra.Command{
					RunE: CreateCmdRunFn(&cfg, vpr, func(cmd *cobra.Command, args []string) error {
						return nil
					}),
				}

				PopulateFlagSet(&cfg, exampleCommand.Flags(), vpr)
				vpr.BindPFlags(exampleCommand.Flags())

				b := bytes.NewBufferString("")
				exampleCommand.SetOut(b)
				exampleCommand.SetArgs([]string{fmt.Sprintf("--config=%s", "testdata/substitutions.yml")})

				err := exampleCommand.Execute()
				Expect(err).ToNot(HaveOccurred())
				Expect(cfg.Foo).To(Equal("test-val-1"))
				Expect(cfg.Foos).To(Equal([]string{"test-val-2", ""}))
				Expect(cfg.Bar).To(Equal(123))
				Expect(cfg.Baz).To(Equal(10 * time.Hour))
				Expect(cfg.FooBar).To(Equal("test-val-4"))
				Expect(cfg.FooFoo).To(Equal(10.23))
				Expect(cfg.FooBytes).To(Equal(100 * bytesize.MB))
				Expect(cfg.FooDur).To(Equal(5*time.Minute + 23*time.Second))
			})

			It("arguments take precedence", func() {
				cfg := FlagsStruct{}
				vpr := viper.New()
				exampleCommand := &cobra.Command{
					RunE: func(cmd *cobra.Command, args []string) error {
						if cfg.Config != "" {
							// Use config file from the flag.
							vpr.SetConfigFile(cfg.Config)

							// If a config file is found, read it in.
							if err := vpr.ReadInConfig(); err == nil {
								fmt.Fprintln(os.Stderr, "Using config file:", vpr.ConfigFileUsed())
							}

							if err := Unmarshal(vpr, &cfg); err != nil {
								fmt.Fprintln(os.Stderr, "Unable to unmarshal:", err)
							}

							fmt.Printf("configuration is %+v \n", cfg)
						}

						return nil
					},
				}

				PopulateFlagSet(&cfg, exampleCommand.Flags(), vpr)
				vpr.BindPFlags(exampleCommand.Flags())

				b := bytes.NewBufferString("")
				exampleCommand.SetOut(b)
				exampleCommand.SetArgs([]string{
					fmt.Sprintf("--config=%s", "testdata/example.yml"),
					fmt.Sprintf("--foo=%s", "test-val-4"),
					fmt.Sprintf("--foo-dur=%s", "3h"),
				})

				err := exampleCommand.Execute()
				Expect(err).ToNot(HaveOccurred())
				Expect(cfg.Foo).To(Equal("test-val-4"))
				Expect(cfg.FooDur).To(Equal(3 * time.Hour))
			})
			It("server configuration", func() {
				var cfg config.Server
				vpr := viper.New()
				exampleCommand := &cobra.Command{
					RunE: CreateCmdRunFn(&cfg, vpr, func(cmd *cobra.Command, args []string) error {
						Expect(loadScrapeConfigsFromFile(&cfg)).ToNot(HaveOccurred())
						fmt.Printf("configuration is %+v \n", cfg)
						return nil
					}),
				}

				PopulateFlagSet(&cfg, exampleCommand.Flags(), vpr, WithSkip("scrape-configs"))
				vpr.BindPFlags(exampleCommand.Flags())

				b := bytes.NewBufferString("")
				exampleCommand.SetOut(b)
				exampleCommand.SetArgs([]string{
					"--config=testdata/server.yml",
					"--log-level=debug",
					"--adhoc-data-path=", // Override as it's platform dependent.
					"--auth.signup-default-role=admin",
				})

				err := exampleCommand.Execute()
				Expect(err).ToNot(HaveOccurred())
				Expect(cfg).To(Equal(config.Server{
					AnalyticsOptOut:         false,
					Config:                  "testdata/server.yml",
					LogLevel:                "debug",
					BadgerLogLevel:          "error",
					StoragePath:             "/var/lib/pyroscope",
					APIBindAddr:             ":4040",
					BaseURL:                 "",
					CacheEvictThreshold:     0.25,
					CacheEvictVolume:        0.33,
					MinFreeSpacePercentage:  5,
					BadgerNoTruncate:        false,
					DisablePprofEndpoint:    false,
					EnableExperimentalAdmin: true,
					NoAdhocUI:               false,
					MaxNodesSerialization:   2048,
					MaxNodesRender:          8192,
					HideApplications:        []string{},
					Retention:               0,
					Database: config.Database{
						Type: "sqlite3",
						URL:  "",
					},
					RetentionLevels: config.RetentionLevels{
						Zero: 100 * time.Second,
						One:  1000 * time.Second,
					},
					SampleRate:          0,
					OutOfSpaceThreshold: 0,
					CacheDimensionSize:  0,
					CacheDictionarySize: 0,
					CacheSegmentSize:    0,
					CacheTreeSize:       0,
					CORS: config.CORSConfig{
						AllowedOrigins: []string{},
						AllowedHeaders: []string{},
						AllowedMethods: []string{},
						MaxAge:         0,
					},
					Auth: config.Auth{
						Internal: config.InternalAuth{
							Enabled:       false,
							SignupEnabled: false,
							AdminUser: config.AdminUser{
								Create:   true,
								Name:     "admin",
								Email:    "admin@localhost.local",
								Password: "admin",
							},
						},
						Google: config.GoogleOauth{
							Enabled:        false,
							ClientID:       "",
							ClientSecret:   "",
							RedirectURL:    "",
							AuthURL:        "https://accounts.google.com/o/oauth2/auth",
							TokenURL:       "https://accounts.google.com/o/oauth2/token",
							AllowedDomains: []string{},
						},
						Gitlab: config.GitlabOauth{
							Enabled:       false,
							ClientID:      "",
							ClientSecret:  "",
							RedirectURL:   "",
							AuthURL:       "https://gitlab.com/oauth/authorize",
							TokenURL:      "https://gitlab.com/oauth/token",
							APIURL:        "https://gitlab.com/api/v4",
							AllowedGroups: []string{},
						},
						Github: config.GithubOauth{
							Enabled:              false,
							ClientID:             "",
							ClientSecret:         "",
							RedirectURL:          "",
							AuthURL:              "https://github.com/login/oauth/authorize",
							TokenURL:             "https://github.com/login/oauth/access_token",
							AllowedOrganizations: []string{},
						},
						Ingestion: config.IngestionAuth{
							Enabled:   false,
							CacheTTL:  time.Second,
							CacheSize: 1024,
						},
						JWTSecret:                "",
						LoginMaximumLifetimeDays: 0,
						SignupDefaultRole:        "admin",
						CookieSameSite:           http.SameSiteStrictMode,
					},

					MetricsExportRules: config.MetricsExportRules{
						"my_metric_name": {
							Expr:    `app.name{foo=~"bar"}`,
							Node:    "a;b;c",
							GroupBy: []string{"foo"},
						},
					},
					AdminSocketPath: "/tmp/pyroscope.sock",

					RemoteWrite: config.RemoteWrite{
						Enabled: true,
					},

					ScrapeConfigs: []*scrape.Config{
						{
							JobName:          "testing",
							EnabledProfiles:  []string{"cpu", "mem"},
							Profiles:         scrape.DefaultConfig().Profiles,
							ScrapeInterval:   10 * time.Second,
							ScrapeTimeout:    15 * time.Second,
							Scheme:           "http",
							HTTPClientConfig: scrape.DefaultHTTPClientConfig,
							ServiceDiscoveryConfigs: []discovery.Config{
								discovery.StaticConfig{
									{
										Targets: []sm.LabelSet{
											{"__address__": "localhost:6060", "__name__": "app", "__spy_name__": ""},
										},
										Labels: sm.LabelSet{"foo": "bar"},
										Source: "0",
									},
								},
							},
						},
					},
				}))
			})

			It("agent configuration", func() {
				var cfg config.Agent
				vpr := viper.New()
				exampleCommand := &cobra.Command{
					Run: func(cmd *cobra.Command, args []string) {
						Expect(vpr.BindPFlags(cmd.Flags())).ToNot(HaveOccurred())
						vpr.SetConfigFile(cfg.Config)
						Expect(vpr.ReadInConfig()).ToNot(HaveOccurred())
						Expect(Unmarshal(vpr, &cfg)).ToNot(HaveOccurred())
						Expect(loadAgentConfig(&cfg)).ToNot(HaveOccurred())
					},
				}

				PopulateFlagSet(&cfg, exampleCommand.Flags(), vpr)
				exampleCommand.SetArgs([]string{
					"--config=testdata/agent.yml",
					"--log-level=debug",
					"--tag=foo=xxx",
				})

				Expect(exampleCommand.Execute()).ToNot(HaveOccurred())
				Expect(cfg).To(Equal(config.Agent{
					Config:                 "testdata/agent.yml",
					LogLevel:               "debug",
					NoLogging:              false,
					ServerAddress:          "http://localhost:4040",
					AuthToken:              "",
					UpstreamThreads:        4,
					UpstreamRequestTimeout: 10 * time.Second,
					Targets: []config.Target{
						{
							ServiceName:        "foo",
							SpyName:            "debugspy",
							ApplicationName:    "foo.app",
							SampleRate:         0,
							DetectSubprocesses: false,
							PyspyBlocking:      false,
							Tags: map[string]string{
								"foo": "xxx",
								"baz": "qux",
							},
						},
					},
					Tags: map[string]string{
						"foo": "xxx",
						"baz": "qux",
					},
				}))
			})
		})
	})
})
