import { API_CONFIG } from './config'
import { HttpClient } from './httpClient'

const aggregationClient = new HttpClient(API_CONFIG.gatewayBaseUrl)

export interface SupportedFiatCurrency {
  code: string
  displayName: string
}

export interface ListSupportedFiatCurrenciesResponse {
  currencies: SupportedFiatCurrency[]
}

export interface FiatLegErrorCandidate {
  coin_id: string
  name: string
}

export interface FiatLegError {
  code: string
  candidates?: FiatLegErrorCandidate[]
}

export interface MoneyLeg {
  symbol: string
  crypto_amount: string
  fiat_amount?: string
  error?: FiatLegError
}

export interface AggregatedTransaction {
  txId: string
  source: string
  importId: string
  timeUtc: string
  kind: string
  inMoney?: MoneyLeg
  outMoney?: MoneyLeg
  feeMoney?: MoneyLeg
  txHash?: string
  note: string
  contractSymbol?: string
  derivativeKind: string
  positionId?: string
  orderId?: string
  txFingerprint: string
}

export interface ListTransactionsByImportResponse {
  transactions: AggregatedTransaction[]
  total: number
}

export interface TenantSettings {
  fiatCurrency: string
  timezone: string
}

export interface UpsertTenantSettingsBody {
  fiatCurrency: string
  timezone: string
}

interface SettingsResponse {
  settings: TenantSettings
}

interface ListTransactionsByImportInput {
  importId: string
  limit?: number
  offset?: number
}

export async function listSupportedFiatCurrencies(): Promise<SupportedFiatCurrency[]> {
  const response = await aggregationClient.request<ListSupportedFiatCurrenciesResponse>('/v1/fiat-currencies', {
    method: 'GET',
  })

  return response.currencies
}

export async function listTransactionsByImport(input: ListTransactionsByImportInput): Promise<ListTransactionsByImportResponse> {
  return aggregationClient.request<ListTransactionsByImportResponse>(
    `/v1/imports/${input.importId}/transactions`,
    {
      method: 'GET',
      query: {
        limit: input.limit,
        offset: input.offset,
      },
    },
  )
}

export async function getTenantSettings(): Promise<TenantSettings> {
  const response = await aggregationClient.request<SettingsResponse>('/v1/settings', {
    method: 'GET',
  })

  return response.settings
}

export async function upsertTenantSettings(body: UpsertTenantSettingsBody): Promise<TenantSettings> {
  const response = await aggregationClient.request<SettingsResponse>('/v1/settings', {
    method: 'PUT',
    body,
  })

  return response.settings
}
