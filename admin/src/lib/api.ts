import { useLogto } from '@logto/vue'
import { config } from '@/config'

// RFC 7807 problem details — what Huma returns on error.
interface ProblemDetail {
  title?: string
  detail?: string
  status?: number
}

/** Error carrying the HTTP status so callers can branch on 401/403/etc. */
export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

/**
 * useApi returns an authenticated fetch bound to the gateway. The bearer token is
 * minted for the Infra API resource (the JWT audience KrakenD validates), so calls
 * carry the scopes the endpoints require (e.g. write:organizations).
 */
export function useApi() {
  const { getAccessToken } = useLogto()

  async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const token = await getAccessToken(config.logtoApiResource)
    const res = await fetch(`${config.apiBaseUrl}${path}`, {
      ...init,
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
        ...init.headers,
      },
    })

    if (!res.ok) {
      let message = res.statusText
      try {
        const body = (await res.json()) as ProblemDetail
        message = body.detail || body.title || message
      } catch {
        // non-JSON error body — keep statusText
      }
      throw new ApiError(message, res.status)
    }

    if (res.status === 204) return undefined as T
    return (await res.json()) as T
  }

  return { request }
}
