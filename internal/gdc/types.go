package gdc

import "encoding/json"

type QueryResponse struct {
	ID             string   `json:"id"`
	CanonicalID    string   `json:"canonical_id"`
	MatchedBy      string   `json:"matched_by"`
	Type           string   `json:"type"`
	Layer          string   `json:"layer"`
	Status         string   `json:"status"`
	Namespace      string   `json:"namespace,omitempty"`
	QualifiedName  string   `json:"qualified_name,omitempty"`
	SpecPath       string   `json:"spec_path,omitempty"`
	ImplPath       string   `json:"impl_path,omitempty"`
	Aliases        []string `json:"aliases,omitempty"`
	Responsibility string   `json:"responsibility"`
	Dependencies   []string `json:"dependencies,omitempty"`
	Methods        []string `json:"methods,omitempty"`
	Properties     []string `json:"properties,omitempty"`
}

type DepsResponse struct {
	Node         string     `json:"node"`
	Depth        int        `json:"depth"`
	Dependencies []DepEntry `json:"dependencies"`
}

type DepEntry struct {
	Target     string         `json:"target"`
	Type       string         `json:"type"`
	Injection  string         `json:"injection"`
	Optional   bool           `json:"optional"`
	Usage      string         `json:"usage"`
	Resolved   bool           `json:"resolved"`
	TargetSpec *DepTargetSpec `json:"target_spec,omitempty"`
}

type DepTargetSpec struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Layer     string `json:"layer"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
}

type RefsResponse struct {
	Node       string     `json:"node"`
	Depth      int        `json:"depth"`
	References []RefEntry `json:"references"`
}

type RefEntry struct {
	Node      string `json:"node"`
	Type      string `json:"type"`
	Layer     string `json:"layer"`
	Namespace string `json:"namespace"`
	DepType   string `json:"dep_type"`
	Injection string `json:"injection"`
	Optional  bool   `json:"optional"`
}

type ContextResponse struct {
	Node           ContextNodeInfo       `json:"node"`
	Responsibility ContextResponsibility `json:"responsibility"`
	Interface      ContextInterface      `json:"interface"`
	Dependencies   []ContextDep          `json:"dependencies"`
	Implementation json.RawMessage       `json:"implementation"`
	Tests          json.RawMessage       `json:"tests"`
	Callers        json.RawMessage       `json:"callers"`
	References     json.RawMessage       `json:"references"`
	Warnings       []string              `json:"warnings"`
}

type ContextNodeInfo struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Layer     string `json:"layer"`
	Namespace string `json:"namespace"`
	FilePath  string `json:"file_path"`
	Status    string `json:"status"`
}

type ContextResponsibility struct {
	Summary    string   `json:"summary"`
	Details    string   `json:"details"`
	Invariants []string `json:"invariants"`
	Boundaries string   `json:"boundaries"`
}

type ContextInterface struct {
	Constructors []ContextConstructor `json:"constructors"`
	Methods      []ContextMethod      `json:"methods"`
	Properties   []ContextProperty    `json:"properties"`
	Events       []ContextEvent       `json:"events"`
}

type ContextConstructor struct {
	Signature   string             `json:"signature"`
	Description string             `json:"description"`
	Parameters  []ContextParameter `json:"parameters"`
}

type ContextMethod struct {
	Name        string             `json:"name"`
	Signature   string             `json:"signature"`
	Description string             `json:"description"`
	Parameters  []ContextParameter `json:"parameters"`
	Returns     ContextReturn      `json:"returns"`
}

type ContextParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ContextReturn struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ContextProperty struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Access      string `json:"access"`
	Description string `json:"description"`
}

type ContextEvent struct {
	Name        string `json:"name"`
	Signature   string `json:"signature"`
	Description string `json:"description"`
}

type ContextDep struct {
	Target    string          `json:"target"`
	Type      string          `json:"type"`
	Injection string          `json:"injection"`
	Optional  bool            `json:"optional"`
	Usage     string          `json:"usage"`
	Depth     int             `json:"depth"`
	Spec      *ContextDepSpec `json:"spec,omitempty"`
}

type ContextDepSpec struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Layer         string `json:"layer"`
	Status        string `json:"status"`
	InterfaceCode string `json:"interface_code,omitempty"`
}

type DiffResponse struct {
	Node     string       `json:"node"`
	FilePath string       `json:"file_path"`
	HasDrift bool         `json:"has_drift"`
	Drift    *DriftReport `json:"drift"`
}

type DriftReport struct {
	MissingMethods      []string         `json:"missing_methods,omitempty"`
	ExtraMethods        []string         `json:"extra_methods,omitempty"`
	MethodMismatches    []SignatureDrift `json:"method_mismatches,omitempty"`
	MissingProperties   []string         `json:"missing_properties,omitempty"`
	ExtraProperties     []string         `json:"extra_properties,omitempty"`
	PropertyMismatches  []TypeDrift      `json:"property_mismatches,omitempty"`
	MissingEvents       []string         `json:"missing_events,omitempty"`
	ExtraEvents         []string         `json:"extra_events,omitempty"`
	EventMismatches     []SignatureDrift `json:"event_mismatches,omitempty"`
	MissingConstructors []string         `json:"missing_constructors,omitempty"`
	ExtraConstructors   []string         `json:"extra_constructors,omitempty"`
	MissingDeps         []string         `json:"missing_deps,omitempty"`
	ExtraDeps           []string         `json:"extra_deps,omitempty"`
}

type SignatureDrift struct {
	Name          string `json:"name"`
	SpecSignature string `json:"spec_signature"`
	CodeSignature string `json:"code_signature"`
}

type TypeDrift struct {
	Name     string `json:"name"`
	SpecType string `json:"spec_type"`
	CodeType string `json:"code_type"`
}

type CheckResponse struct {
	Issues  []CheckIssue `json:"issues"`
	Summary CheckSummary `json:"summary"`
	Result  string       `json:"result"`
}

type CheckIssue struct {
	Severity   string `json:"severity"`
	Category   string `json:"category"`
	SourceNode string `json:"source_node,omitempty"`
	TargetNode string `json:"target_node,omitempty"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

type CheckSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Info     int `json:"info"`
	Total    int `json:"total"`
}

type GraphResponse struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type GraphNode struct {
	ID     string   `json:"id"`
	Type   string   `json:"type"`
	Layer  string   `json:"layer"`
	Status string   `json:"status"`
	Tags   []string `json:"tags,omitempty"`
}

type GraphEdge struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Type      string `json:"type"`
	Optional  bool   `json:"optional,omitempty"`
	Violation bool   `json:"violation,omitempty"`
}
