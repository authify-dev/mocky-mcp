package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Create a new MCP server
	s := server.NewMCPServer(
		"Demo 🚀",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Tool definition
	tool := mcp.NewTool("hello_world",
		mcp.WithDescription("Say hello to someone via FastAPI"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the person to greet"),
		),
	)
	// Add handler
	s.AddTool(tool, helloHandler)

	listTool := mcp.NewTool("list_prototypes",
		mcp.WithDescription("Fetch prototypes and return a list of {id, name, method, urlPath}"),
	)
	s.AddTool(listTool, listPrototypesHandler)

	// Tool definition
	// Tool 3: get_prototype_detail (nuevo)
	detailTool := mcp.NewTool("get_prototype_detail",
		mcp.WithDescription("Get prototype detail by id or name. If name is provided, it will be resolved to an id first."),
		mcp.WithString("id", mcp.DefaultString(""), mcp.Description("Prototype id (preferred if known)")),
		mcp.WithString("name", mcp.DefaultString(""), mcp.Description("Prototype name (if id not provided)")),
	)
	s.AddTool(detailTool, getPrototypeDetailHandler)

	// Start stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// Struct para decodificar la respuesta del API
type HelloResponse struct {
	Message string `json:"message"`
}

func helloHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Devuelve un saludo directamente, sin llamar a la API
	message := fmt.Sprintf("¡Hola, %s! 👋", name)
	return mcp.NewToolResultText(message), nil
}

// ====== Tool 2: list_prototypes ======

// Estructuras para parsear la respuesta del endpoint de Mocky
type mockyResponse struct {
	Results []mockyItem `json:"results"`
}

type mockyItem struct {
	ID      string       `json:"id"`
	Name    string       `json:"name"`
	Request mockyRequest `json:"request"`
}

type mockyRequest struct {
	Method  string `json:"method"`
	URLPath string `json:"urlPath"`
}

func listPrototypesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	const url = "https://development.jalocompany.tech/mocky/v1/prototypes"

	resp, err := http.Get(url)
	if err != nil {
		return mcp.NewToolResultError("Error calling prototypes API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError("Error reading prototypes response: " + err.Error()), nil
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return mcp.NewToolResultError(fmt.Sprintf("Non-OK status %d from prototypes API: %s", resp.StatusCode, string(body))), nil
	}

	var data mockyResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return mcp.NewToolResultError("Error parsing prototypes JSON: " + err.Error()), nil
	}

	// Preparamos la salida compacta para el LLM con los campos solicitados
	type outItem struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Method  string `json:"method"`
		URLPath string `json:"urlPath"`
	}

	out := make([]outItem, 0, len(data.Results))
	for _, it := range data.Results {
		out = append(out, outItem{
			ID:      it.ID,
			Name:    it.Name,
			Method:  it.Request.Method,
			URLPath: it.Request.URLPath,
		})
	}

	// Lo devolvemos como JSON (texto) para que el LLM lo liste claramente
	pretty, _ := json.MarshalIndent(out, "", "  ")
	return mcp.NewToolResultText(string(pretty)), nil
}

// ====== Tool 3: get_prototype_detail ======

// Estructuras del detalle
type detailEnvelope struct {
	Data detailItem `json:"data"`
	// status_code, success, trace_id existen pero no son críticos para la salida
}

type detailItem struct {
	CreatedAt string             `json:"createdAt"`
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Request   mockyRequestDetail `json:"request"`
	Response  detailResponse     `json:"response"`
	UpdatedAt string             `json:"updatedAt"`
}

type mockyRequestDetail struct {
	BodySchema any               `json:"bodySchema"`
	Delay      int               `json:"delay"`
	Headers    map[string]string `json:"headers"`
	Method     string            `json:"method"`
	PathParams map[string]any    `json:"path_params"`
	URLPath    string            `json:"urlPath"`
}

type detailResponse struct {
	Body struct {
		Data       map[string]any `json:"data"`
		StatusCode int            `json:"status_code"`
		Success    bool           `json:"success"`
	} `json:"body"`
}

func getPrototypeDetailHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	const base = "http://209.126.13.207:8010/v1/prototypes"

	// Leer argumentos opcionales
	id := request.GetString("id", "")
	name := request.GetString("name", "")

	// Resolver por nombre, si no hay id
	if id == "" {
		if name == "" {
			return mcp.NewToolResultError("Provide either 'id' or 'name'."), nil
		}
		resolvedID, err := resolvePrototypeIDByName(name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		id = resolvedID
	}

	// Llamar detalle por id
	detailURL := fmt.Sprintf("%s/%s", base, id)
	resp, err := http.Get(detailURL)
	if err != nil {
		return mcp.NewToolResultError("Error calling prototype detail API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError("Error reading prototype detail response: " + err.Error()), nil
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return mcp.NewToolResultError(
			fmt.Sprintf("Non-OK status %d from detail API: %s", resp.StatusCode, string(body))), nil
	}

	var envelop detailEnvelope
	if err := json.Unmarshal(body, &envelop); err != nil {
		return mcp.NewToolResultError("Error parsing detail JSON: " + err.Error()), nil
	}

	// Salida compacta con los campos clave
	type detailOut struct {
		ID                string            `json:"id"`
		Name              string            `json:"name"`
		Method            string            `json:"method"`
		URLPath           string            `json:"urlPath"`
		Delay             int               `json:"delay"`
		ResponseStatus    int               `json:"response_status_code"`
		ResponseSuccess   bool              `json:"response_success"`
		ResponseData      map[string]any    `json:"response_data"`
		CreatedAt         string            `json:"createdAt"`
		UpdatedAt         string            `json:"updatedAt"`
		AdditionalHeaders map[string]string `json:"headers,omitempty"`
	}

	out := detailOut{
		ID:                envelop.Data.ID,
		Name:              envelop.Data.Name,
		Method:            envelop.Data.Request.Method,
		URLPath:           envelop.Data.Request.URLPath,
		Delay:             envelop.Data.Request.Delay,
		ResponseStatus:    envelop.Data.Response.Body.StatusCode,
		ResponseSuccess:   envelop.Data.Response.Body.Success,
		ResponseData:      envelop.Data.Response.Body.Data,
		CreatedAt:         envelop.Data.CreatedAt,
		UpdatedAt:         envelop.Data.UpdatedAt,
		AdditionalHeaders: envelop.Data.Request.Headers,
	}

	pretty, _ := json.MarshalIndent(out, "", "  ")
	return mcp.NewToolResultText(string(pretty)), nil
}

// Helper: resuelve ID por nombre (case-insensitive)
func resolvePrototypeIDByName(name string) (string, error) {
	const url = "http://209.126.13.207:8010/v1/prototypes"

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("error calling prototypes API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading prototypes response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("non-OK status %d from prototypes API: %s", resp.StatusCode, string(body))
	}

	var data mockyResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("error parsing prototypes JSON: %w", err)
	}

	var candidates []string
	target := strings.TrimSpace(strings.ToLower(name))
	for _, it := range data.Results {
		candidates = append(candidates, it.Name)
		if strings.ToLower(it.Name) == target {
			return it.ID, nil
		}
	}

	return "", fmt.Errorf("prototype with name %q not found. Available: %v", name, candidates)
}
