import { NextResponse } from "next/server"

/**
 * Legacy login route - redirects to /login which uses Clerk SignIn component.
 * Clerk handles OAuth provider selection natively.
 */
export async function GET(request: Request) {
  const { searchParams, origin } = new URL(request.url)
  const returnTo = searchParams.get("returnTo")

  const loginUrl = new URL("/login", origin)
  if (returnTo) {
    loginUrl.searchParams.set("returnTo", returnTo)
  }

  return NextResponse.redirect(loginUrl)
}
