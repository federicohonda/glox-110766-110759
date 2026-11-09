sub fib {
    my $n = shift;
    return $n if $n < 2;
    return fib($n - 1) + fib($n - 2);
}
print fib(28), "\n";
