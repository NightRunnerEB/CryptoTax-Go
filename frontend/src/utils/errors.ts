import { ApiError } from '../api/httpClient'

interface FieldViolation {
  field?: string
  description?: string
}

interface BadRequestDetail {
  '@type'?: string
  fieldViolations?: FieldViolation[]
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object'
}

function isBadRequestDetail(detail: unknown): detail is BadRequestDetail {
  return isRecord(detail) && detail['@type'] === 'type.googleapis.com/google.rpc.BadRequest'
}

function extractFieldViolations(details: unknown): string[] {
  if (!Array.isArray(details)) {
    return []
  }

  const violations: string[] = []

  details.forEach((detail) => {
    if (!isBadRequestDetail(detail) || !Array.isArray(detail.fieldViolations)) {
      return
    }

    detail.fieldViolations.forEach((item) => {
      const field = typeof item.field === 'string' ? item.field.trim() : ''
      const description = typeof item.description === 'string' ? item.description.trim() : ''

      if (field && description) {
        violations.push(`${field}: ${description}`)
        return
      }

      if (description) {
        violations.push(description)
      }
    })
  })

  return violations
}

export function toErrorMessages(error: unknown, fallback = 'Unexpected error'): string[] {
  if (error instanceof ApiError) {
    const messages: string[] = []
    const base = error.message.trim()
    if (base) {
      messages.push(base)
    }

    const violationMessages = extractFieldViolations(error.details)
    violationMessages.forEach((item) => {
      if (!messages.includes(item)) {
        messages.push(item)
      }
    })

    return messages.length > 0 ? messages : [fallback]
  }

  if (error instanceof Error) {
    const message = error.message.trim()
    return message ? [message] : [fallback]
  }

  return [fallback]
}

export function toErrorMessage(error: unknown, fallback = 'Unexpected error'): string {
  return toErrorMessages(error, fallback).join('. ')
}
