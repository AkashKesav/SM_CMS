# Email Configuration for Password Reset

**Problem:** DigitalOcean blocks outbound SMTP ports (25, 587, 465), so traditional SMTP email services won't work.

**Solution:** Use email APIs that work over HTTPS (port 443).

## Choose ONE of the following email providers:

### Option 1: SendGrid (Recommended)
- Free tier: 100 emails/day
- Sign up: https://sendgrid.com/
- Add to `.env`:
```bash
SENDGRID_API_KEY=SG.your_api_key_here
SENDGRID_FROM_EMAIL=your-email@example.com
SENDGRID_FROM_NAME=Your App Name
```

### Option 2: Mailgun
- Free tier: 5,000 emails/month
- Sign up: https://www.mailgun.com/
- Add to `.env`:
```bash
MAILGUN_API_KEY=your_api_key_here
MAILGUN_DOMAIN=mg.yourdomain.com
MAILGUN_FROM_EMAIL=your-email@mg.yourdomain.com
```

### Option 3: Brevo (formerly Sendinblue)
- Free tier: 300 emails/day
- Sign up: https://www.brevo.com/
- Add to `.env`:
```bash
BREVO_API_KEY=your_api_key_here
BREVO_FROM_EMAIL=your-email@example.com
BREVO_FROM_NAME=Your App Name
```

## Development Mode (No Email Configured)

If you want to test password reset without email, set:
```bash
RETURN_RESET_LINK=true
```

The reset link will be returned in the API response. **Do NOT use in production.**

## Quick Setup for SendGrid:

1. Create account at https://signup.sendgrid.com/
2. Go to Settings > API Keys
3. Create API Key with "Mail Send" permissions
4. Verify "Single Sender" identity (use your email)
5. Add the API key to your `.env` file

## After Adding Environment Variables

Restart your backend service:
```bash
# On DigitalOcean
sudo systemctl restart cms-backend
# or if using docker compose
docker-compose restart backend
```

## Testing

To test password reset:
```bash
curl -X POST https://your-domain.com/api/auth/forgot-password \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}'
```
