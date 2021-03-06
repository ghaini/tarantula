<h1 align="center"> Tarantula - Go HTTP toolkit </h1>

[![Go Report Card](https://goreportcard.com/badge/github.com/ghaini/tarantula)](https://goreportcard.com/report/github.com/ghaini/tarantula)
[![GoDoc](https://godoc.org/github.com/ghaini/tarantula?status.svg)](https://godoc.org/github.com/ghaini/tarantula)
[![LICENSE](https://img.shields.io/github/license/ghaini/tarantula.svg?style=flat-square)](https://github.com/ghaini/tarantula/blob/master/LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/ghaini/tarantula)](https://github.com/ghaini/tarantula/stargazers)
[![GitHub issues](https://img.shields.io/github/issues/ghaini/tarantula)](https://github.com/ghaini/tarantula/issues)
[![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/ghaini/tarantula/issues)
[![GitHub Release](https://img.shields.io/github/release/ghaini/tarantula)](https://github.com/ghaini/tarantula/releases)


 tarantula is a fast and multi-purpose HTTP toolkit allow to run multiple probers
 
### Installation:

    go get github.com/ghaini/tarantula
    
### Usage:

    t := tarantulas.NewTarantulas()
    t.MultiThread(100)                          // optional - default: 1 thread
    t.SetTimeout(15)                            // optional - default: 5 seconds
    t.SetPorts([]int{443,80,8080})              // optional - default: 80,443
    t.SetRetry(5)                               // on failure request - optional - default: 80,443
    t.SetUserAgents([]string{"curl"})           // use custom user agent - optional  
    t.HTTPProxy("proxy.com:80")                 // use http proxy for requests (if you have socks proxy, you can use t.SocksProxy()) - optional
    t.GetContents(domain, []string{subdomains}) // receive active assets
    
### Documentation:

The <a href="https://github.com/ghaini/tarantula/wiki">wiki</a> contains all the documentation related to Tarantula.
    
### Bugs and feature requests:

Bugs and feature request are tracked on <a href="https://github.com/ghaini/tarantula/issues">GitHub</a>

### License:

Tarantula is under the Apache 2.0 license. See the <a href="https://github.com/ghaini/tarantula/blob/master/LICENSE">LICENSE</a> file for details.

