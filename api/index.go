// Copyright 2023 igevin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stulzq/azure-openai-proxy/azure"
	"log"
	"net/http"
	"os"
)

var (
	version   = ""
	buildDate = ""
	gitCommit = ""
)

var router *gin.Engine

func init() {
	viper.AutomaticEnv()
	parseFlag()

	err := azure.Init()
	if err != nil {
		panic(err)
	}

	gin.SetMode(gin.ReleaseMode)
	router = gin.Default()

	// if viper get cors is true, then apply corsMiddleware
	if viper.GetBool("cors") {
		log.Printf("CORS supported! \n")
		router.Use(corsMiddleware())
	}

	registerRoute(router)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	router.ServeHTTP(w, r)
	//fmt.Fprintf(w, "<h1>Hello from Go!</h1>")
}

// corsMiddleware sets up the CORS headers for all responses
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Clear any previously set headers
		if c.Request.Method != "POST" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, X-Stainless-OS, X-STAINLESS-LANG, X-STAINLESS-PACKAGE-VERSION, X-STAINLESS-RUNTIME, X-STAINLESS-RUNTIME-VERSION, X-STAINLESS-ARCH")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(200)
			return
		}
		c.Next()
	}
}

func parseFlag() {
	pflag.StringP("configFile", "c", "config.yaml", "config file")
	pflag.StringP("listen", "l", ":8080", "listen address")
	pflag.BoolP("version", "v", false, "version information")
	pflag.BoolP("cors", "s", false, "cors support")
	pflag.Parse()
	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		panic(err)
	}
	if viper.GetBool("version") {
		fmt.Println("version:", version)
		fmt.Println("buildDate:", buildDate)
		fmt.Println("gitCommit:", gitCommit)
		os.Exit(0)
	}
}

// registerRoute registers all routes
func registerRoute(r *gin.Engine) {
	// https://platform.openai.com/docs/api-reference
	r.HEAD("/", func(c *gin.Context) {
		c.Status(200)
	})
	r.GET("/#/", func(c *gin.Context) {
		c.String(200, "Hello World")
	})
	r.Any("/health", func(c *gin.Context) {
		c.Status(200)
	})
	apiBase := viper.GetString("api_base")
	stripPrefixConverter := azure.NewStripPrefixConverter(apiBase)
	r.GET(stripPrefixConverter.Prefix+"/models", azure.ModelProxy)
	templateConverter := azure.NewTemplateConverter("/openai/deployments/{{.DeploymentName}}/embeddings")
	apiBasedRouter := r.Group(apiBase)
	{
		apiBasedRouter.Any("/engines/:model/embeddings", azure.ProxyWithConverter(templateConverter))
		apiBasedRouter.Any("/completions", azure.ProxyWithConverter(stripPrefixConverter))
		apiBasedRouter.Any("/chat/completions", azure.ProxyWithConverter(stripPrefixConverter))
		apiBasedRouter.Any("/embeddings", azure.ProxyWithConverter(stripPrefixConverter))
	}
}
