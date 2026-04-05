import { clearSession, getSession } from '../auth/sessionStore'
import { runRefreshFlow } from '../auth/refreshCoordinator'

type PrimitiveQueryValue = string | number | boolean

type QueryValue = PrimitiveQueryValue | null | undefined

interface RequestQuery {
  [key: string]: QueryValue
}

export interface RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  query?: RequestQuery
  body?: FormData | unknown
  headers?: HeadersInit
  auth?: boolean
  refreshOnUnauthorized?: boolean
  signal?: AbortSignal
}

export class ApiError extends Error {
  readonly status: number
  readonly code?: number
  readonly details?: unknown
  readonly payload?: unknown

  constructor(params: { status: number; message: string; code?: number; details?: unknown; payload?: unknown }) {
    super(params.message)
    this.name = 'ApiError'
    this.status = params.status
    this.code = params.code
    this.details = params.details
    this.payload = params.payload
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object'
}

function parseJsonSafely(rawText: string): unknown {
  try {
    return JSON.parse(rawText) as unknown
  } catch {
    return rawText
  }
}

function resolveErrorMessage(status: number, payload: unknown): string {
  if (isRecord(payload)) {
    if (typeof payload.message === 'string' && payload.message.trim() !== '') {
      return payload.message
    }

    if (typeof payload.error === 'string' && payload.error.trim() !== '') {
      return payload.error
    }
  }

  if (typeof payload === 'string' && payload.trim() !== '') {
    return payload
  }

  return `Request failed with status ${status}`
}

function resolveErrorCode(payload: unknown): number | undefined {
  if (!isRecord(payload) || typeof payload.code !== 'number') {
    return undefined
  }

  return payload.code
}

function resolveErrorDetails(payload: unknown): unknown {
  if (!isRecord(payload)) {
    return undefined
  }

  return payload.details
}

function isAccessTokenNearExpiry(): boolean {
  const session = getSession()
  if (!session) {
    return false
  }

  const bufferMs = 30_000
  return session.tokens.accessExpiresAt - Date.now() <= bufferMs
}

export class HttpClient {
  private readonly baseUrl: string

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl
  }

  async request<TResponse>(path: string, options: RequestOptions = {}): Promise<TResponse> {
    const {
      method = 'GET',
      query,
      body,
      headers,
      auth = true,
      refreshOnUnauthorized = true,
      signal,
    } = options

    if (auth && refreshOnUnauthorized && isAccessTokenNearExpiry()) {
      await runRefreshFlow()
    }

    const execute = async (): Promise<Response> => {
      const requestUrl = new URL(path, `${this.baseUrl}/`)
      if (query) {
        Object.entries(query).forEach(([key, value]) => {
          if (value === null || value === undefined) {
            return
          }
          requestUrl.searchParams.set(key, String(value))
        })
      }

      const requestHeaders = new Headers(headers)
      if (!requestHeaders.has('Accept')) {
        requestHeaders.set('Accept', 'application/json')
      }

      if (auth) {
        const session = getSession()
        if (session?.tokens.accessToken) {
          requestHeaders.set('Authorization', `Bearer ${session.tokens.accessToken}`)
        }
      }

      let requestBody: BodyInit | undefined
      if (body instanceof FormData) {
        requestBody = body
      } else if (body !== undefined) {
        requestHeaders.set('Content-Type', 'application/json')
        requestBody = JSON.stringify(body)
      }

      try {
        return await fetch(requestUrl.toString(), {
          method,
          headers: requestHeaders,
          body: requestBody,
          signal,
        })
      } catch (error) {
        const reason = error instanceof Error ? error.message : 'unknown network error'
        throw new Error(`Network error calling ${method} ${requestUrl.toString()}: ${reason}`)
      }
    }

    let response = await execute()

    if (response.status === 401 && auth && refreshOnUnauthorized) {
      const refreshed = await runRefreshFlow()
      if (refreshed) {
        response = await execute()
      } else {
        clearSession()
      }
    }

    return parseResponse<TResponse>(response)
  }
}

async function parseResponse<TResponse>(response: Response): Promise<TResponse> {
  if (response.status === 204) {
    return undefined as TResponse
  }

  const rawText = await response.text()
  const payload = rawText.length === 0 ? undefined : parseJsonSafely(rawText)

  if (!response.ok) {
    throw new ApiError({
      status: response.status,
      code: resolveErrorCode(payload),
      details: resolveErrorDetails(payload),
      message: resolveErrorMessage(response.status, payload),
      payload,
    })
  }

  return payload as TResponse
}
