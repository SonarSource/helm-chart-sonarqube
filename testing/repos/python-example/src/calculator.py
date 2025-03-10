class Calculator:
    """
    A simple calculator class that provides basic arithmetic operations
    """

    def add(self, a: float, b: float) -> float:
        """
        Add two numbers together

        Args:
            a: First number
            b: Second number

        Returns:
            Sum of a and b
        """
        return a + b

    def subtract(self, a: float, b: float) -> float:
        """
        Subtract second number from first number

        Args:
            a: First number
            b: Second number

        Returns:
            Difference between a and b
        """
        return a - b

    def multiply(self, a: float, b: float) -> float:
        """
        Multiply two numbers

        Args:
            a: First number
            b: Second number

        Returns:
            Product of a and b
        """
        return a * b

    def divide(self, a: float, b: float) -> float:
        """
        Divide first number by second number

        Args:
            a: First number
            b: Second number

        Returns:
            Quotient of a divided by b

        Raises:
            ValueError: If b is zero
        """
        if b == 0:
            raise ValueError("Cannot divide by zero")
        return a / b
