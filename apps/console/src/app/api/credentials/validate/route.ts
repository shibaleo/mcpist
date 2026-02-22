import { NextRequest, NextResponse } from 'next/server'
import { auth } from '@clerk/nextjs/server'
import { validators, requiredParams, ValidationResult } from './validators'

export async function POST(request: NextRequest) {
  try {
    const { userId } = await auth()
    if (!userId) {
      return NextResponse.json(
        { valid: false, error: 'Unauthorized' },
        { status: 401 }
      )
    }

    const body = await request.json()
    const { service, token, ...extra } = body

    if (!service || !token) {
      return NextResponse.json(
        { valid: false, error: 'service と token が必要です' },
        { status: 400 }
      )
    }

    // Check required params for this service
    const required = requiredParams[service] || []
    for (const param of required) {
      if (!extra[param]) {
        return NextResponse.json(
          { valid: false, error: `${service}には${param}が必要です` },
          { status: 400 }
        )
      }
    }

    // Get validator for this service
    const validator = validators[service]

    let result: ValidationResult
    if (validator) {
      result = await validator({ token, ...extra })
    } else {
      // Skip validation for OAuth 2.0 modules
      result = { valid: true }
    }

    return NextResponse.json(result)
  } catch {
    return NextResponse.json(
      { valid: false, error: 'リクエストの処理に失敗しました' },
      { status: 500 }
    )
  }
}
