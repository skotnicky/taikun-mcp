# Taikun MCP Server Development Guidelines

## Project Overview

This project provides an MCP (Model Context Protocol) server for Taikun Cloud Platform, enabling AI assistants to interact with Taikun's infrastructure management capabilities through structured tools.

## Resources

### Official Taikun Resources
- **Taikun API Documentation**: https://api.taikun.cloud/swagger/
- **Taikun Showback API**: https://api.taikun.cloud/showback/swagger/
- **Taikun Platform**: https://docs.taikun.cloud/

### Go Client Library
- **Taikun Go Client**: https://github.com/itera-io/taikungoclient
  - Auto-generated nightly from OpenAPI specs
  - Used by this project for all API interactions
  - Contains all API models and client methods
  - Example usage: `client.Client.CatalogAPI.CatalogList(ctx)`

### Terraform Provider (Reference Implementation)
- **Terraform Provider Taikun**: https://github.com/itera-io/terraform-provider-taikun
  - Excellent reference for API usage patterns
  - Shows how to handle Taikun API responses
  - Resource implementations demonstrate proper field handling
  - Error handling patterns to follow

### MCP Protocol
- **MCP Specification**: https://modelcontextprotocol.io/
- **MCP Go SDK**: https://github.com/metoro-io/mcp-golang
  - Used for tool registration and responses
  - Provides structured communication with AI assistants

### Development Tools
- **Go Documentation**: Use `go doc github.com/itera-io/taikungoclient/client` to explore API types
- **API Inspection**: Create temporary Go files to inspect struct fields when needed
- **Taikun CLI**: May provide additional usage examples

### Common API Patterns to Reference

#### Boolean Parameters
- **ALWAYS call boolean setters unconditionally**: Throughout the codebase, boolean setters (e.g., `SetForceDeleteVClusters(args.ForceDeleteVClusters)`) must be called with their actual value (true or false) rather than being wrapped in an `if` block. This ensures the API receives the user-provided value rather than falling back to an internal default.

#### Catalog Operations
- **Adding Apps**: See `terraform-provider-taikun/taikun/catalog/resource_taikun_catalog.go`
- **Listing**: Most list operations follow similar patterns in the Terraform provider
- **Validation**: Check Terraform provider for input validation patterns

#### Project Management
- **Virtual Clusters**: See `terraform-provider-taikun/taikun/project/` directory
- **Project Apps**: Application deployment patterns in Terraform provider

#### Authentication
- **Client Creation**: `taikungoclient.NewClientFromCredentials(username, password, "", "", "", apiHost)`
- **Environment Variables**: `TAIKUN_EMAIL`, `TAIKUN_PASSWORD`, `TAIKUN_API_HOST`

### Troubleshooting Resources

#### API Field Investigation
When unsure about API response fields:
1. Check the Terraform provider implementation
2. Use `go doc` to explore struct definitions
3. Create temporary inspection files to examine field types
4. Refer to Swagger documentation for field descriptions

#### Common Issues
- **Nullable Fields**: Always check `.IsSet()` and `.Get() != nil` for `NullableString` types
- **Pointer Fields**: Check for `!= nil` before dereferencing
- **Pagination**: Most list APIs support `limit`, `offset`, and `search` parameters
- **Error Handling**: **ALWAYS use `createError()` for ALL Taikun API errors** - provides clear, detailed error messages from the API. Use `ErrorResponse` only for custom validation errors

## Project Structure

### File Organization
- **`main.go`**: Tool registration, MCP server setup, response helpers
- **`catalogs.go`**: Catalog and catalog app management tools
- **`applications.go`**: Project application lifecycle tools
- **`projects.go`**: Project listing and management tools
- **`virtualclusters.go`**: Virtual cluster (project) creation and management
- **`basic_test.go`**: Compilation and JSON marshaling tests
- **`CLAUDE.md`**: This file - development guidelines

### Adding New Tool Categories
When adding new functionality that doesn't fit existing files:
1. Create a new `.go` file (e.g., `storage.go`, `networking.go`)
2. Follow the same import pattern as existing files
3. Add argument structs to `main.go` if they're shared
4. Update `basic_test.go` with new argument types
5. Register tools in `main.go`

### Integration Testing
- **`test-integration.sh`**: Integration test script
- **`test_mcp.sh`**: MCP-specific testing script
- Set up proper environment variables before running tests

## Code Standards

### JSON Response Format
**ALL tools must return JSON responses, never plain text.**

#### Required Response Types
- **Success operations**: Use `SuccessResponse` struct with `createJSONResponse()`
- **Error responses**: Use `ErrorResponse` struct with `createJSONResponse()`  
- **List operations**: Return structured JSON with arrays of typed objects
- **Detail operations**: Return structured JSON objects with all relevant fields

#### Examples

**✅ CORRECT - Success Response:**
```go
successResp := SuccessResponse{
    Message: "Operation completed successfully",
    Success: true,
}
return createJSONResponse(successResp), nil
```

**✅ CORRECT - List Response:**
```go
listResp := struct {
    Items   []ItemType `json:"items"`
    Total   int        `json:"total"`
    Message string     `json:"message"`
}{
    Items:   items,
    Total:   len(items),
    Message: "Found X items",
}
return createJSONResponse(listResp), nil
```

**❌ INCORRECT - Plain Text:**
```go
// NEVER DO THIS
return mcp_golang.NewToolResponse(
    mcp_golang.NewTextContent("Operation completed"),
), nil
```

#### Error Handling
- **ALWAYS use `createError()` for ALL Taikun API errors** - this uses `taikungoclient.CreateError()` internally
- The `createError()` function provides clear, detailed error messages from the Taikun API
- For custom validation errors (non-API), use `ErrorResponse` struct:
```go
// ✅ CORRECT - For API errors
if err != nil {
    return createError(response, err), nil
}

// ✅ CORRECT - For custom validation errors
errorResp := ErrorResponse{
    Error: "Custom validation error message",
}
return createJSONResponse(errorResp), nil
```

**Why use `createError()`?**
- Provides detailed Taikun API error messages
- Handles HTTP response codes properly
- Extracts meaningful error details from API responses
- Maintains consistent error formatting across all tools

**The `createError()` Helper Function:**
```go
// Located in main.go - uses taikungoclient.CreateError internally
func createError(response *http.Response, err error) *mcp_golang.ToolResponse {
    // Use taikungoclient's CreateError for detailed error messages
    taikunErr := taikungoclient.CreateError(response, err)
    
    var errorResp ErrorResponse
    if taikunErr != nil {
        errorResp.Error = taikunErr.Error()
    } else {
        errorResp.Error = "Unknown error occurred"
    }

    logger.Printf("Error occurred: %s", errorResp.Error)
    return createJSONResponse(errorResp)
}
```

**Example of Clear Error Messages:**
- **Without `createError()`**: `"HTTP error 404"` (unclear)
- **With `createError()`**: `"Taikun Error: wordpress not found (HTTP 404)"` (clear and actionable)

This is why we **ALWAYS** use `createError()` for Taikun API errors!

### Code Organization

#### Function Structure
1. Context creation
2. API request building with parameters
3. API execution
4. Error handling with `createError()` and `checkResponse()`
5. Data transformation to structured types
6. JSON response creation with `createJSONResponse()`

#### Type Definitions
- Define response structs with proper JSON tags
- Use `omitempty` for optional fields
- Group related types together
- Use meaningful field names

### Testing Requirements

#### Add New Tools to Tests
When adding new tools, update `basic_test.go`:
```go
{
    name: "YourNewToolArgs",
    data: YourNewToolArgs{
        // test data
    },
},
```

#### Compilation Verification
- Run `go test -v` to verify compilation
- Run `go build` to ensure no build errors
- Check for unused imports

### API Client Usage

#### Nullable Fields
Handle Taikun API nullable fields properly:
```go
// For NullableString
if field.IsSet() && field.Get() != nil {
    result.Field = *field.Get()
}

// For pointer fields
if field != nil {
    result.Field = *field
}
```

#### Pagination Support
Always support standard pagination parameters:
```go
if args.Limit > 0 {
    req = req.Limit(args.Limit)
}
if args.Offset > 0 {
    req = req.Offset(args.Offset)
}
if args.Search != "" {
    req = req.Search(args.Search)
}
```

### Tool Registration

#### Registration Pattern
```go
err = server.RegisterTool("tool-name", "Clear description", func(args ToolArgs) (*mcp_golang.ToolResponse, error) {
    client := taikungoclient.NewClientFromCredentials(username, password, "", "", "", apiHost)
    return toolFunction(client, args)
})
if err != nil {
    logger.Fatalf("Failed to register tool-name tool: %v", err)
}
logger.Println("Registered tool-name tool")
```

### Documentation

#### Tool Descriptions
- Use clear, concise descriptions
- Follow kebab-case naming (e.g., `list-available-packages`)
- Include parameter descriptions in struct tags

#### JSON Schema
Always include proper jsonschema tags:
```go
type ToolArgs struct {
    Field string `json:"field" jsonschema:"required,description=Field description"`
}
```

## Development Workflow

1. **Design**: Plan the tool's JSON response structure first
2. **Implement**: Write the function with proper error handling
3. **Test**: Add to `basic_test.go` and verify compilation
4. **Register**: Add tool registration with proper error handling
5. **Validate**: Ensure all responses are JSON formatted

## Quality Checklist

Before submitting any new tool:
- [ ] All responses use `createJSONResponse()`
- [ ] **ALL Taikun API errors use `createError()` helper** (provides clear error messages)
- [ ] Custom validation errors use `ErrorResponse` struct
- [ ] Struct types defined with proper JSON tags
- [ ] Added to `basic_test.go`
- [ ] `go test -v` passes
- [ ] `go build` succeeds
- [ ] No unused imports
- [ ] Tool registered with proper error handling
- [ ] Documentation includes clear descriptions