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

export interface ListTransactionsResponse {
  items: AggregatedTransaction[]
  nextPageToken: string
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

interface ListTransactionsInput {
  pageSize?: number
  pageToken?: string
  dateFrom?: string
  dateTo?: string
  importId?: string
  source?: string
  kind?: string
  targetFiat?: string
}

export async function listSupportedFiatCurrencies(): Promise<SupportedFiatCurrency[]> {
  const response = await aggregationClient.request<ListSupportedFiatCurrenciesResponse>('/fiat-currencies', {
    method: 'GET',
  })

  return response.currencies
}

export async function listTransactions(input: ListTransactionsInput): Promise<ListTransactionsResponse> {
  return aggregationClient.request<ListTransactionsResponse>(
    '/transactions',
    {
      method: 'GET',
      query: {
        page_size: input.pageSize,
        page_token: input.pageToken,
        date_from: input.dateFrom,
        date_to: input.dateTo,
        import_id: input.importId,
        source: input.source,
        kind: input.kind,
        target_fiat: input.targetFiat,
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
