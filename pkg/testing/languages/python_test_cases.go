package languages

import "github.com/MacAttak/pi-scanner/pkg/detection"

// PythonTestCases returns comprehensive test cases for Python code scanning
func PythonTestCases() []MultiLanguageTestCase {
	return []MultiLanguageTestCase{
		// TRUE POSITIVES - Real PI in Python code
		{
			ID:          "python-tfn-001",
			Language:    "python",
			Filename:    "user_service.py",
			Code:        `class UserService:\n    DEFAULT_TFN = "123456782"  # Valid TFN\n    \n    def __init__(self):\n        self.tfn = self.DEFAULT_TFN`,
			ExpectedPI:  true,
			PIType:      detection.PITypeTFN,
			Context:     "production",
			Rationale:   "Hardcoded valid TFN in Python class",
		},
		{
			ID:          "python-medicare-001",
			Language:    "python",
			Filename:    "patient_controller.py",
			Code:        `from flask import Flask, request, jsonify\n\nclass PatientController:\n    def create_patient(self):\n        patient_data = {\n            'medicare_number': '2428778132'\n        }\n        return jsonify(patient_data)`,
			ExpectedPI:  true,
			PIType:      detection.PITypeMedicare,
			Context:     "production",
			Rationale:   "Valid Medicare number in Flask controller",
		},
		{
			ID:          "python-abn-001",
			Language:    "python",
			Filename:    "company_model.py",
			Code:        `from django.db import models\n\nclass Company(models.Model):\n    name = models.CharField(max_length=100)\n    abn = models.CharField(max_length=11, default="51824753556")  # Commonwealth Bank ABN`,
			ExpectedPI:  true,
			PIType:      detection.PITypeABN,
			Context:     "production",
			Rationale:   "Valid ABN in Django model",
		},
		{
			ID:          "python-bsb-001",
			Language:    "python",
			Filename:    "banking_service.py",
			Code:        `class BankingService:
    CBA_BSB = "062-001"  # Commonwealth Bank BSB
    ANZ_BSB = "013-006"  # ANZ Bank BSB
    
    def validate_bsb(self, bsb):
        return len(bsb.replace("-", "")) == 6`,
			ExpectedPI:  true,
			PIType:      detection.PITypeBSB,
			Context:     "production",
			Rationale:   "Valid BSB numbers in Python banking service",
		},
		{
			ID:          "python-acn-001",
			Language:    "python",
			Filename:    "company_views.py",
			Code:        `from django.http import JsonResponse\nfrom django.views import View\n\nclass CompanyView(View):\n    def get(self, request):\n        # Company ACN: 123456780\n        company = self.get_company_by_acn("123456780")\n        return JsonResponse({"company": company})`,
			ExpectedPI:  true,
			PIType:      detection.PITypeACN,
			Context:     "production",
			Rationale:   "Valid ACN in Django view",
		},

		// FALSE POSITIVES - Code constructs that look like names
		{
			ID:          "python-false-name-001",
			Language:    "python",
			Filename:    "user_service.py",
			Code:        `class UserService:\n    def __init__(self):\n        self.data_processor = DataProcessor()\n        self.http_client = HttpClient()`,
			ExpectedPI:  false,
			PIType:      detection.PITypeName,
			Context:     "production",
			Rationale:   "Class and attribute names, not person names",
		},
		{
			ID:          "python-false-name-002",
			Language:    "python",
			Filename:    "security_config.py",
			Code:        `from django.conf import settings\n\nclass SecurityConfig:\n    def get_auth_manager(self):\n        return CustomAuthManager()`,
			ExpectedPI:  false,
			PIType:      detection.PITypeName,
			Context:     "production",
			Rationale:   "Django configuration class and method names",
		},
		{
			ID:          "python-false-name-003",
			Language:    "python",
			Filename:    "payment_service.py",
			Code:        `import requests\nimport json\n\nclass PaymentService:\n    def __init__(self):\n        self.rest_template = requests.Session()\n        self.json_parser = json\n        self.error_handler = ErrorHandler()`,
			ExpectedPI:  false,
			PIType:      detection.PITypeName,
			Context:     "production",
			Rationale:   "Python technical component names, not person names",
		},
		{
			ID:          "python-false-name-004",
			Language:    "python",
			Filename:    "stream_processor.py",
			Code:        `class StreamProcessor:\n    def __init__(self):\n        self.event_handler = EventHandler()\n        self.message_queue = MessageQueue()\n        self.data_transformer = DataTransformer()`,
			ExpectedPI:  false,
			PIType:      detection.PITypeName,
			Context:     "production",
			Rationale:   "Stream processing component names",
		},

		// TEST CONTEXTS - Valid PI but in test files (should be filtered)
		{
			ID:          "python-test-tfn-001",
			Language:    "python",
			Filename:    "test_user_service.py",
			Code:        `import unittest\n\nclass TestUserService(unittest.TestCase):\n    def test_validate_tfn(self):\n        test_tfn = "123456782"\n        self.assertTrue(validator.is_valid(test_tfn))`,
			ExpectedPI:  false,
			PIType:      detection.PITypeTFN,
			Context:     "test",
			Rationale:   "Valid TFN but in test file context",
		},
		{
			ID:          "python-mock-medicare-001",
			Language:    "python",
			Filename:    "test_data_factory.py",
			Code:        `class TestDataFactory:\n    MOCK_MEDICARE = "2428778132"\n    \n    @staticmethod\n    def create_test_patient():\n        return Patient(medicare_number=TestDataFactory.MOCK_MEDICARE)`,
			ExpectedPI:  false,
			PIType:      detection.PITypeMedicare,
			Context:     "test",
			Rationale:   "Mock data for testing, not production PI",
		},

		// EDGE CASES
		{
			ID:          "python-pytest-param-001",
			Language:    "python",
			Filename:    "test_validation.py",
			Code:        `import pytest\n\n@pytest.mark.parametrize("tfn", ["123456782", "876543217"])\ndef test_tfn_validation(tfn):\n    assert TFNValidator.is_valid(tfn)`,
			ExpectedPI:  false,
			PIType:      detection.PITypeTFN,
			Context:     "test",
			Rationale:   "Test data in pytest parameters, not production usage",
		},
		{
			ID:          "python-comment-pi-001",
			Language:    "python",
			Filename:    "user_service.py",
			Code:        `class UserService:\n    # Example TFN format: 123456782\n    def validate_tfn(self, tfn):\n        return bool(re.match(r'\\d{9}', tfn))`,
			ExpectedPI:  false,
			PIType:      detection.PITypeTFN,
			Context:     "documentation",
			Rationale:   "TFN in comment for documentation purposes",
		},

		// LOGGING CONCERNS
		{
			ID:          "python-logging-pi-001",
			Language:    "python",
			Filename:    "audit_service.py",
			Code:        `import logging\n\nclass AuditService:\n    def __init__(self):\n        self.logger = logging.getLogger(__name__)\n    \n    def log_user_action(self, user_id, tfn):\n        self.logger.info(f"User {user_id} accessed TFN: {tfn}")  # TFN: 123456782`,
			ExpectedPI:  true,
			PIType:      detection.PITypeTFN,
			Context:     "logging",
			Rationale:   "TFN being logged - security risk even if example",
		},

		// DJANGO/FLASK PATTERNS
		{
			ID:          "python-django-view-001",
			Language:    "python",
			Filename:    "customer_views.py",
			Code:        `from django.http import JsonResponse\nfrom django.views import View\n\nclass CustomerView(View):\n    def post(self, request):\n        customer = {\n            'name': 'John Smith',\n            'tfn': '123456782',\n            'email': 'john.smith@example.com'\n        }\n        return JsonResponse(customer)`,
			ExpectedPI:  true,
			PIType:      detection.PITypeTFN,
			Context:     "production",
			Rationale:   "Multiple PI types in Django view - critical risk",
		},

		// CONFIGURATION FILES
		{
			ID:          "python-config-pi-001",
			Language:    "python",
			Filename:    "settings.py",
			Code:        `# Django settings\nDEBUG = True\nDATABASE_URL = "postgresql://user:pass@localhost/db"\n\n# Demo data\nDEMO_TFN = "123456782"\nDEMO_MEDICARE = "2428778132"`,
			ExpectedPI:  true,
			PIType:      detection.PITypeTFN,
			Context:     "configuration",
			Rationale:   "Demo PI data in configuration - potential exposure",
		},

		// DATA SCIENCE / PANDAS PATTERNS
		{
			ID:          "python-pandas-pi-001",
			Language:    "python",
			Filename:    "data_processor.py",
			Code:        `import pandas as pd\n\ndef process_customer_data():\n    df = pd.DataFrame({\n        'name': ['John Smith', 'Jane Doe'],\n        'tfn': ['123456782', '987654321'],\n        'medicare': ['2428778132', '3456789012']\n    })\n    return df`,
			ExpectedPI:  true,
			PIType:      detection.PITypeTFN,
			Context:     "production",
			Rationale:   "PI data in pandas DataFrame - data processing risk",
		},

		// ASYNC/AWAIT PATTERNS
		{
			ID:          "python-async-pi-001",
			Language:    "python",
			Filename:    "async_user_service.py",
			Code:        `import asyncio\n\nclass AsyncUserService:\n    async def validate_user(self, tfn="123456782"):\n        # Async validation with default TFN\n        return await self.external_validator.validate(tfn)`,
			ExpectedPI:  true,
			PIType:      detection.PITypeTFN,
			Context:     "production",
			Rationale:   "Default TFN parameter in async function",
		},

		// MULTI-PI SCENARIOS
		{
			ID:          "python-multi-pi-001",
			Language:    "python",
			Filename:    "customer_dto.py",
			Code:        `from dataclasses import dataclass\n\n@dataclass\nclass CustomerDto:\n    full_name: str = "John Smith"\n    tfn: str = "123456782"\n    medicare: str = "2428778132"\n    email: str = "john.smith@example.com"\n    address: str = "123 Collins St, Melbourne VIC 3000"`,
			ExpectedPI:  true,
			PIType:      detection.PITypeTFN, // Multiple types, use primary
			Context:     "production",
			Rationale:   "Multiple PI types in dataclass - critical risk",
		},

		// ENVIRONMENT VARIABLES
		{
			ID:          "python-env-pi-001",
			Language:    "python",
			Filename:    "config.py",
			Code:        `import os\n\nclass Config:\n    # Fallback to hardcoded values if env vars not set\n    TEST_TFN = os.getenv('TEST_TFN', '123456782')\n    TEST_MEDICARE = os.getenv('TEST_MEDICARE', '2428778132')`,
			ExpectedPI:  true,
			PIType:      detection.PITypeTFN,
			Context:     "configuration",
			Rationale:   "Hardcoded PI fallback values in configuration",
		},
	}
}