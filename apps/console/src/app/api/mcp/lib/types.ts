/**
 * MCP Protocol Types
 * JSON-RPC 2.0 based Message Control Protocol
 */

// JSON-RPC 2.0 Request
export interface McpRequest {
  jsonrpc: '2.0'
  id: string | number
  method: string
  params?: Record<string, unknown>
}

// JSON-RPC 2.0 Response
export interface McpResponse {
  jsonrpc: '2.0'
  id: string | number
  result?: unknown
  error?: McpError
}

// JSON-RPC 2.0 Error
export interface McpError {
  code: number
  message: string
  data?: unknown
}

// MCP Error Codes
export const MCP_ERROR_CODES = {
  PARSE_ERROR: -32700,
  INVALID_REQUEST: -32600,
  METHOD_NOT_FOUND: -32601,
  INVALID_PARAMS: -32602,
  INTERNAL_ERROR: -32603,
} as const

// Tool Definition (JSON Schema based)
export interface ToolDefinition {
  name: string
  description: string
  inputSchema: {
    type: 'object'
    properties: Record<string, unknown>
    required?: string[]
  }
}

// Tool Handler Function Type
export type ToolHandler = (
  params: Record<string, unknown>,
  context: ToolContext
) => Promise<ToolResult>

// Tool Context (passed to handlers)
export interface ToolContext {
  userId: string
}

// Tool Result
export interface ToolResult {
  content: Array<{
    type: 'text'
    text: string
  }>
  isError?: boolean
}

// Module Definition (for registry)
export interface ModuleDefinition {
  name: string
  description: string
  tools: ToolDefinition[]
  handlers: Record<string, ToolHandler>
}

// Authentication Result
export interface AuthResult {
  userId: string | null
  error?: string
  method?: 'jwt' | 'mcp_token' | 'service_role'
}

// MCP Protocol Methods
export type McpMethod =
  | 'initialize'
  | 'tools/list'
  | 'tools/call'
  | 'ping'
