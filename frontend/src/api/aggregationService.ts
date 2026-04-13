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

export interface UserSettings {
  fiatCurrency: string
  timezone: string
}

export interface UpsertUserSettingsBody {
  fiatCurrency: string
  timezone: string
}

interface SettingsResponse {
  settings: UserSettings
}

interface ListTransactionsByImportInput {
  importId: string
  limit?: number
  offset?: number
}

export async function listSupportedFiatCurrencies(): Promise<SupportedFiatCurrency[]> {
  const response = await aggregationClient.request<ListSupportedFiatCurrenciesResponse>('/fiat-currencies', {
    method: 'GET',
  })

  return response.currencies
}

export async function listTransactionsByImport(input: ListTransactionsByImportInput): Promise<ListTransactionsByImportResponse> {
  return aggregationClient.request<ListTransactionsByImportResponse>(
    `/imports/${input.importId}/transactions`,
    {
      method: 'GET',
      query: {
        limit: input.limit,
        offset: input.offset,
      },
    },
  )
}

export async function getUserSettings(): Promise<UserSettings> {
  const response = await aggregationClient.request<SettingsResponse>('/settings', {
    method: 'GET',
  })

  return response.settings
}

export async function upsertUserSettings(body: UpsertUserSettingsBody): Promise<UserSettings> {
  const response = await aggregationClient.request<SettingsResponse>('/settings', {
    method: 'PUT',
    body,
  })

  return response.settings
}
