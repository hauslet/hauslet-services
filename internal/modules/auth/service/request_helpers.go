package service

// StoreRequestMetadata stores IP and User-Agent for a login request
// This is used to populate session data in enrichClaims
func (s *AuthServiceImpl) StoreRequestMetadata(email, ip, userAgent string) {
	s.requestMetadata.Set(email, ip, userAgent)
}
