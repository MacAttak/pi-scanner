package languages

import "github.com/MacAttak/pi-scanner/pkg/detection"

// ScalaTestCases returns comprehensive test cases for Scala code scanning
func ScalaTestCases() []MultiLanguageTestCase {
	return []MultiLanguageTestCase{
		// TRUE POSITIVES - Real PI in Scala code
		{
			ID:          "scala-tfn-001",
			Language:    "scala",
			Filename:    "UserService.scala",
			Code:        `object UserService {\n  val DEFAULT_TFN = "123456782" // Valid TFN\n}`,
			ExpectedPI:  true,
			PIType:      detection.PITypeTFN,
			Context:     "production",
			Rationale:   "Hardcoded valid TFN in Scala object",
		},
		{
			ID:          "scala-medicare-001",
			Language:    "scala",
			Filename:    "PatientController.scala",
			Code:        `class PatientController @Inject()(cc: ControllerComponents) extends AbstractController(cc) {\n  def createPatient = Action(parse.json) { request =>\n    val patient = Patient(medicareNumber = "2428778132")\n    Ok(Json.toJson(patient))\n  }\n}`,
			ExpectedPI:  true,
			PIType:      detection.PITypeMedicare,
			Context:     "production",
			Rationale:   "Valid Medicare number in Play Framework controller",
		},
		{
			ID:          "scala-abn-001",
			Language:    "scala",
			Filename:    "CompanyEntity.scala",
			Code:        `case class Company(\n  id: Long,\n  name: String,\n  abn: String = "51824753556" // Commonwealth Bank ABN\n)`,
			ExpectedPI:  true,
			PIType:      detection.PITypeABN,
			Context:     "production",
			Rationale:   "Valid ABN in Scala case class",
		},
		{
			ID:          "scala-bsb-001",
			Language:    "scala",
			Filename:    "BankingService.scala",
			Code:        `object BankingService {\n  val CBA_BSB = "062-001" // Commonwealth Bank BSB\n  val WBC_BSB = "032-001" // Westpac BSB\n  \n  def validateBSB(bsb: String): Boolean = bsb.matches("\\\\d{3}-\\\\d{3}")\n}`,
			ExpectedPI:  true,
			PIType:      detection.PITypeBSB,
			Context:     "production",
			Rationale:   "Valid BSB numbers in Scala banking service",
		},
		{
			ID:          "scala-acn-001",
			Language:    "scala",
			Filename:    "CompanyController.scala",
			Code:        `class CompanyController @Inject()(cc: ControllerComponents) extends AbstractController(cc) {\n  def getCompany = Action { request =>\n    // Company ACN: 123456780\n    val company = companyService.findByACN("123456780")\n    Ok(Json.toJson(company))\n  }\n}`,
			ExpectedPI:  true,
			PIType:      detection.PITypeACN,
			Context:     "production",
			Rationale:   "Valid ACN in Play Framework controller",
		},

		// FALSE POSITIVES - Code constructs that look like names
		{
			ID:          "scala-false-name-001",
			Language:    "scala",
			Filename:    "UserService.scala",
			Code:        `trait UserService {\n  val dataProcessor: DataProcessor\n  val httpClient: HttpClient\n}`,
			ExpectedPI:  false,
			PIType:      detection.PITypeName,
			Context:     "production",
			Rationale:   "Trait and field names, not person names",
		},
		{
			ID:          "scala-false-name-002",
			Language:    "scala",
			Filename:    "SecurityConfig.scala",
			Code:        `@Configuration\nclass SecurityConfig {\n  def authManager: AuthenticationManager = new CustomAuthManager()\n}`,
			ExpectedPI:  false,
			PIType:      detection.PITypeName,
			Context:     "production",
			Rationale:   "Spring configuration class and method names",
		},
		{
			ID:          "scala-false-name-003",
			Language:    "scala",
			Filename:    "PaymentService.scala",
			Code:        `class PaymentService {\n  private val restTemplate: RestTemplate = _\n  private val jsonParser: JsonParser = _\n  private val errorHandler: ErrorHandler = _\n}`,
			ExpectedPI:  false,
			PIType:      detection.PITypeName,
			Context:     "production",
			Rationale:   "Scala technical component names, not person names",
		},
		{
			ID:          "scala-false-name-004",
			Language:    "scala",
			Filename:    "StreamProcessor.scala",
			Code:        `object StreamProcessor {\n  val eventHandler: EventHandler = EventHandler()\n  val messageQueue: MessageQueue = MessageQueue()\n  val dataTransformer: DataTransformer = DataTransformer()\n}`,
			ExpectedPI:  false,
			PIType:      detection.PITypeName,
			Context:     "production",
			Rationale:   "Akka/streaming framework component names",
		},

		// TEST CONTEXTS - Valid PI but in test files (should be filtered)
		{
			ID:          "scala-test-tfn-001",
			Language:    "scala",
			Filename:    "UserServiceSpec.scala",
			Code:        `class UserServiceSpec extends WordSpec with Matchers {\n  "UserService" should {\n    "validate TFN" in {\n      val testTFN = "123456782"\n      validator.isValid(testTFN) shouldBe true\n    }\n  }\n}`,
			ExpectedPI:  false,
			PIType:      detection.PITypeTFN,
			Context:     "test",
			Rationale:   "Valid TFN but in test file context",
		},
		{
			ID:          "scala-mock-medicare-001",
			Language:    "scala",
			Filename:    "TestDataFactory.scala",
			Code:        `object TestDataFactory {\n  val MOCK_MEDICARE = "2428778132"\n  def createTestPatient(): Patient = Patient(medicareNumber = MOCK_MEDICARE)\n}`,
			ExpectedPI:  false,
			PIType:      detection.PITypeMedicare,
			Context:     "test",
			Rationale:   "Mock data for testing, not production PI",
		},

		// EDGE CASES
		{
			ID:          "scala-pattern-match-pi-001",
			Language:    "scala",
			Filename:    "ValidationService.scala",
			Code:        `def validateTFN(tfn: String): Boolean = tfn match {\n  case "123456782" | "876543217" => true\n  case _ => TFNValidator.isValid(tfn)\n}`,
			ExpectedPI:  false,
			PIType:      detection.PITypeTFN,
			Context:     "test",
			Rationale:   "Test data in pattern matching, not production usage",
		},
		{
			ID:          "scala-comment-pi-001",
			Language:    "scala",
			Filename:    "UserService.scala",
			Code:        `class UserService {\n  // Example TFN format: 123456782\n  def validateTFN(tfn: String): Boolean = {\n    tfn.matches("\\\\d{9}")\n  }\n}`,
			ExpectedPI:  false,
			PIType:      detection.PITypeTFN,
			Context:     "documentation",
			Rationale:   "TFN in comment for documentation purposes",
		},

		// LOGGING CONCERNS
		{
			ID:          "scala-logging-pi-001",
			Language:    "scala",
			Filename:    "AuditService.scala",
			Code:        `class AuditService {\n  private val logger = LoggerFactory.getLogger(classOf[AuditService])\n  \n  def logUserAction(userId: String, tfn: String): Unit = {\n    logger.info(s"User $userId accessed TFN: $tfn") // TFN: 123456782\n  }\n}`,
			ExpectedPI:  true,
			PIType:      detection.PITypeTFN,
			Context:     "logging",
			Rationale:   "TFN being logged - security risk even if example",
		},

		// FUNCTIONAL PROGRAMMING PATTERNS
		{
			ID:          "scala-functional-pi-001",
			Language:    "scala",
			Filename:    "ValidationService.scala",
			Code:        `object ValidationService {\n  val validTFNs = List("123456782", "987654321", "456789123")\n  \n  def validateTFN(tfn: String): Either[ValidationError, ValidTFN] = {\n    if (validTFNs.contains(tfn)) Right(ValidTFN(tfn))\n    else Left(InvalidTFN(tfn))\n  }\n}`,
			ExpectedPI:  true,
			PIType:      detection.PITypeTFN,
			Context:     "production",
			Rationale:   "Valid TFNs in production validation logic",
		},

		// AKKA/SPRAY PATTERNS
		{
			ID:          "scala-akka-pi-001",
			Language:    "scala",
			Filename:    "UserActor.scala",
			Code:        `class UserActor extends Actor {\n  import UserActor._\n  \n  def receive = {\n    case CreateUser(name, tfn) if tfn == "123456782" =>\n      sender() ! UserCreated(User(name, tfn))\n    case _ =>\n      sender() ! InvalidRequest\n  }\n}`,
			ExpectedPI:  true,
			PIType:      detection.PITypeTFN,
			Context:     "production",
			Rationale:   "TFN validation in Akka actor - potential hardcoded value",
		},

		// MULTI-PI SCENARIOS
		{
			ID:          "scala-multi-pi-001",
			Language:    "scala",
			Filename:    "CustomerDto.scala",
			Code:        `case class CustomerDto(\n  fullName: String = "John Smith",\n  tfn: String = "123456782",\n  medicare: String = "2428778132",\n  email: String = "john.smith@example.com",\n  address: String = "123 Collins St, Melbourne VIC 3000"\n)`,
			ExpectedPI:  true,
			PIType:      detection.PITypeTFN, // Multiple types, use primary
			Context:     "production",
			Rationale:   "Multiple PI types in case class - critical risk",
		},

		// IMPLICITS AND TYPE CLASSES
		{
			ID:          "scala-implicit-pi-001",
			Language:    "scala",
			Filename:    "TFNImplicits.scala",
			Code:        `object TFNImplicits {\n  implicit val defaultTFN: TFN = TFN("123456782")\n  \n  implicit class TFNOps(tfn: String) {\n    def isValidTFN: Boolean = TFNValidator.validate(tfn)\n  }\n}`,
			ExpectedPI:  true,
			PIType:      detection.PITypeTFN,
			Context:     "production",
			Rationale:   "Implicit default TFN value - potential data leak",
		},
	}
}