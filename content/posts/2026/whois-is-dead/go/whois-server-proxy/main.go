package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"text/template"
	"time"

	_ "embed"
)

//go:embed rdap.template
var rdapTemplateContent string

// tldRdapServers provides a direct mapping for common TLDs to their RDAP servers.
var tldRdapServers = map[string]string{
	"com": "https://rdap.verisign.com/com/v1/domain/",
	"net": "https://rdap.verisign.com/net/v1/domain/",
	"org": "https://rdap.publicinterestregistry.org/rdap/domain/",
	"dev": "https://pubapi.registry.google/rdap/domain/",
}

// --- RDAP Data Structures ---

type RDAPLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
	Type string `json:"type"`
}

type RDAPEvent struct {
	Action string    `json:"eventAction"`
	Actor  string    `json:"eventActor"`
	Date   time.Time `json:"eventDate"`
}

type RDAPNameserver struct {
	LDHName string `json:"ldhName"`
	Handle  string `json:"handle"`
	Remarks []struct {
		Title       string   `json:"title"`
		Type        string   `json:"type"`
		Description []string `json:"description"`
	} `json:"remarks"`
	Links []RDAPLink `json:"links"`
}

type RDAPEntity struct {
	VCardArray VCard        `json:"vcardArray"`
	Roles      []string     `json:"roles"`
	Entities   []RDAPEntity `json:"entities"`
	Handle     string       `json:"handle"`
	PublicIDs  []struct {
		Type       string `json:"type"`
		Identifier string `json:"identifier"`
	} `json:"publicIds"`
	Remarks []struct {
		Title       string   `json:"title"`
		Type        string   `json:"type"`
		Description []string `json:"description"`
	} `json:"remarks"`
	Links  []RDAPLink `json:"links"`
	Status []string   `json:"status"`
}

type VCard []interface{}

func (vc VCard) GetField(key string) string {
	if len(vc) < 2 {
		return ""
	}
	properties, ok := vc[1].([]interface{})
	if !ok {
		return ""
	}
	for _, prop := range properties {
		propertyArray, ok := prop.([]interface{})
		if !ok || len(propertyArray) < 4 {
			continue
		}
		propKey, ok := propertyArray[0].(string)
		if !ok || propKey != key {
			continue
		}
		val, ok := propertyArray[3].(string)
		if ok {
			return val
		}
	}
	return ""
}

type SecureDNSData struct {
	Algorithm  int    `json:"algorithm"`
	Digest     string `json:"digest"`
	DigestType int    `json:"digestType"`
	KeyTag     int    `json:"keyTag"`
}

type RDAPResponse struct {
	LDHName     string           `json:"ldhName"`
	Handle      string           `json:"handle"`
	Nameservers []RDAPNameserver `json:"nameservers"`
	Events      []RDAPEvent      `json:"events"`
	Entities    []RDAPEntity     `json:"entities"`
	Links       []RDAPLink       `json:"links"`
	Status      []string         `json:"status"`
	Conformance []string         `json:"rdapConformance"`
	Notices     []struct {
		Title       string     `json:"title"`
		Description []string   `json:"description"`
		Links       []RDAPLink `json:"links"`
	} `json:"notices"`
	SecureDNS struct {
		ZoneSigned       bool            `json:"zoneSigned"`
		DelegationSigned bool            `json:"delegationSigned"`
		DSData           []SecureDNSData `json:"dsData"`
	} `json:"secureDNS"`
	Remarks []struct {
		Title       string   `json:"title"`
		Description []string `json:"description"`
	} `json:"remarks"`
}

func (r *RDAPResponse) getReferralURL() string {
	for _, link := range r.Links {
		if link.Rel == "related" && link.Type == "application/rdap+json" {
			return link.Href
		}
	}
	return ""
}

// Server holds the dependencies for the WHOIS server.
type Server struct {
	rdapTemplate *template.Template
}

// NewServer creates a new server and parses the RDAP template.
func NewServer() (*Server, error) {
	tmpl, err := template.New("rdap").Parse(rdapTemplateContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	return &Server{rdapTemplate: tmpl}, nil
}

// queryRDAP performs the RDAP lookup, following one level of referral if necessary.
func queryRDAP(domain string) (*RDAPResponse, error) {
	parts := strings.Split(domain, ".")
	var url string
	if len(parts) > 1 {
		tld := parts[len(parts)-1]
		if baseUrl, ok := tldRdapServers[tld]; ok {
			url = baseUrl + domain
			log.Printf("Found direct RDAP server for TLD .%s, using: %s", tld, url)
		}
	}

	if url == "" {
		url = "https://rdap.iana.org/domain/" + domain
		log.Printf("Using IANA bootstrap RDAP endpoint: %s", url)
	}

	// Perform the initial query
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("RDAP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("RDAP server returned status %d: %s", resp.StatusCode, string(body))
	}

	var initialResponse RDAPResponse
	if err := json.NewDecoder(resp.Body).Decode(&initialResponse); err != nil {
		return nil, fmt.Errorf("failed to decode initial RDAP JSON: %w", err)
	}
	resp.Body.Close() // Close the body of the first response now.

	// Check for a referral and follow it
	if referralURL := initialResponse.getReferralURL(); referralURL != "" {
		log.Printf("Following RDAP referral to: %s", referralURL)
		resp, err = http.Get(referralURL)
		if err != nil {
			return nil, fmt.Errorf("RDAP referral request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("RDAP referral server returned status %d: %s", resp.StatusCode, string(body))
		}

		var finalResponse RDAPResponse
		if err := json.NewDecoder(resp.Body).Decode(&finalResponse); err != nil {
			return nil, fmt.Errorf("failed to decode final RDAP JSON: %w", err)
		}
		return &finalResponse, nil
	}

	return &initialResponse, nil
}

func (s *Server) handleConnection(conn net.Conn) {
	log.Printf("new connection from %s", conn.RemoteAddr())
	defer log.Printf("connection to %s closed", conn.RemoteAddr())
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	var clientQuery string

	if scanner.Scan() {
		clientQuery = strings.TrimSpace(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from client: %v", err)
		return
	}

	if clientQuery == "" {
		_, _ = fmt.Fprint(conn, "Please provide a domain name.\r\n")
		return
	}

	log.Printf("Received query for: %q", clientQuery)

	rdapData, err := queryRDAP(clientQuery)
	if err != nil {
		log.Printf("RDAP query for %q failed: %v", clientQuery, err)
		_, _ = fmt.Fprintf(conn, "Error performing RDAP lookup: %v\r\n", err)
		return
	}

	var responseBuilder strings.Builder
	if err := s.rdapTemplate.Execute(&responseBuilder, rdapData); err != nil {
		log.Printf("Internal error: failed to execute template: %v", err)
		_, _ = fmt.Fprint(conn, "Internal server error.\r\n")
		return
	}

	_, err = fmt.Fprint(conn, responseBuilder.String())
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func main() {
	port := ":43"

	server, err := NewServer()
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}

	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Error listening on port %s: %v", port, err)
	}
	defer listener.Close()
	log.Printf("RDAP Proxy WHOIS server listening on %s", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go server.handleConnection(conn)
	}
}
