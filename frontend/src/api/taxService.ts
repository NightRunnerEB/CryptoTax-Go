import { API_CONFIG } from './config'
import { HttpClient } from './httpClient'

const taxClient = new HttpClient(API_CONFIG.gatewayBaseUrl)

export interface TaxPolicy {
  treatCryptoCryptoAsDisposal: boolean
  costBasisMethod: 'FIFO' | 'LIFO' | 'AVG'
  jurisdiction: string
}

export interface TaxProfile {
  inn: string
  oktmo: string
  lastName: string
  firstName: string
  middleName: string
  timezone: string
  phone: string
  wallets: string[]
  taxResidencyStatus: string
  taxpayerType: string
}

export interface TaxProfileInput {
  inn: string
  oktmo: string
  lastName: string
  firstName: string
  middleName: string
  timezone: string
  phone: string
  wallets: string[]
  taxResidencyStatus: string
  taxpayerType: string
}

interface TaxProfileResponse {
  profile: TaxProfile
}

export interface StartReportParams {
  taxYear: number
  taxPolicy: TaxPolicy
}

export interface StartReportResponse {
  reportId: string
  status: string
}

export interface TaxSummary {
  totalIncomeFiat: string
  totalExpenseFiat: string
  taxBaseFiat: string
  taxDueFiat: string
}

export interface TaxJob {
  reportId: string
  taxYear: number
  policySnapshot: TaxPolicy
  status: string
  attempts: number
  createdAt?: string
  startedAt?: string
  finishedAt?: string
  lastErrorCode?: string
  lastErrorMessage?: string
  auditZipUrl?: string
  reportUrl?: string
  summary?: TaxSummary
}

interface GetReportStatusResponse {
  job: TaxJob
}

interface ListReportsResponse {
  jobs: TaxJob[]
  total: number
}

interface ListReportsInput {
  limit?: number
  offset?: number
}

export async function getTaxProfile(): Promise<TaxProfile> {
  const response = await taxClient.request<TaxProfileResponse>('/tax/profile', {
    method: 'GET',
  })

  return response.profile
}

export async function upsertTaxProfile(profile: TaxProfileInput): Promise<TaxProfile> {
  const response = await taxClient.request<TaxProfileResponse>('/tax/profile', {
    method: 'PUT',
    body: profile,
  })

  return response.profile
}

export async function startTaxReport(params: StartReportParams): Promise<StartReportResponse> {
  return taxClient.request<StartReportResponse>('/tax/reports:start', {
    method: 'POST',
    body: params,
  })
}

export async function listTaxReports(input: ListReportsInput): Promise<ListReportsResponse> {
  return taxClient.request<ListReportsResponse>('/tax/reports', {
    method: 'GET',
    query: {
      limit: input.limit,
      offset: input.offset,
    },
  })
}

export async function getTaxReportStatus(reportId: string): Promise<TaxJob> {
  const response = await taxClient.request<GetReportStatusResponse>(`/tax/reports/${reportId}`, {
    method: 'GET',
  })

  return response.job
}
