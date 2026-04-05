import { API_CONFIG } from './config'
import { HttpClient } from './httpClient'

const ledgerClient = new HttpClient(API_CONFIG.gatewayBaseUrl)

export interface ParseSuccessResponse {
  status: string
}

export interface SupportedExchangesResponse {
  exchanges: string[]
}

export async function listSupportedExchanges(): Promise<string[]> {
  const response = await ledgerClient.request<SupportedExchangesResponse>('/v1/exchanges/supported', {
    method: 'GET',
  })

  return response.exchanges.map((exchange) => exchange.trim().toLowerCase()).filter((exchange) => exchange.length > 0)
}

function resolveUploadPath(exchange: string): string {
  switch (exchange) {
    case 'mexc':
      return '/mexc/csv'
    default:
      throw new Error(`Upload endpoint for exchange "${exchange}" is not defined in frontend API contract.`)
  }
}

export async function uploadExchangeCsv(
  params: {
    file: File
    exchange: string
  },
): Promise<ParseSuccessResponse> {
  const formData = new FormData()
  formData.set('file', params.file)

  return ledgerClient.request<ParseSuccessResponse>(resolveUploadPath(params.exchange), {
    method: 'POST',
    body: formData,
  })
}
