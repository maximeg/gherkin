package Gherkin::Exceptions {
    use Moose;
    extends 'Throwable::Error';
}

# Parent of single and composite exceptions
package Gherkin::Exceptions::Parser {
    use Moose;
    extends 'Gherkin::Exceptions';
}

# Composite exceptions
package Gherkin::Exceptions::CompositeParser {
    use Moose;
    extends 'Gherkin::Exceptions::Parser';

    has '+message' => (
        is         => 'ro',
        required   => 0,
        lazy_build => 1,
    );

    has 'errors' => (
        is      => 'ro',
        isa     => 'ArrayRef[Gherkin::Exceptions::Parser]',
        default => sub { [] },
    );

    around BUILDARGS => sub {
        my ( $orig, $class, @errors ) = @_;
        $class->$orig({ errors => \@errors });
    };

    sub _build_message {
        my $self = shift;
        return join "\n",
            ( 'Parser errors:', map { $_->message } @{ $self->errors } );
    }
}

#
# Various non-composite exceptions
#
package Gherkin::Exceptions::SingleParser {
    use Moose;
    extends 'Gherkin::Exceptions::Parser';

    has 'original_message' => ( is => 'ro', isa => 'Str',     required => 1 );
    has 'location'         => ( is => 'ro', isa => 'HashRef', required => 1 );

    has '+message' => (
        is         => 'ro',
        required   => 0,
        lazy_build => 1,
    );

    around throw => sub {
        my ( $orig, $class, $message, $location ) = @_;
        $class->$orig(
            {   location         => $location,
                original_message => $message,
            }
        );
    };

    sub _build_message {
        my $self = shift;
        return sprintf( '(%i:%i): %s',
            $self->location->{'line'},
            $self->location->{'column'} || '0',
            $self->original_message, );
    }
}

package Gherkin::Exceptions::NoSuchLanguage {
    use Moose;
    extends 'Gherkin::Exceptions::SingleParser';

    around throw => sub {
        my ( $orig, $class, $language, $location ) = @_;
        $class->$orig('Language not supported: ' . $language, $location);
    };
}

package Gherkin::Exceptions::AstBuilder {
    use Moose;
    extends 'Gherkin::Exceptions::SingleParser';
}

package Gherkin::Exceptions::UnexpectedEOF {
    use Moose;
    extends 'Gherkin::Exceptions::SingleParser';

    around throw => sub {
        my ( $orig, $class, $received_token, $expected_token_types )
            = @_;
        $class->$orig(  'unexpected end of file, expected: '
                    . ( join ', ', @$expected_token_types ), $received_token->location );
    };
}

package Gherkin::Exceptions::UnexpectedToken {
    use Moose;
    extends 'Gherkin::Exceptions::SingleParser';

    around throw => sub {
        my ( $orig, $class, $received_token, $expected_token_types,
            $state_comment )
            = @_;

        my $received_token_value = $received_token->token_value;
        $received_token_value =~ s/^\s+//;
        $received_token_value =~ s/\s+$//;

        my $message = "expected: " . ( join ', ', @$expected_token_types ) .
            ", got '$received_token_value'";

        my %location = %{ $received_token->location };
        $location{'column'} = $received_token->line->indent + 1
            unless defined $location{'column'};

        $class->$orig( $message, \%location );
    };
}

1;